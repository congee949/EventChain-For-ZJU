import * as grpc from '@grpc/grpc-js';
import { connect, signers } from '@hyperledger/fabric-gateway';
import fs from 'node:fs';
import crypto from 'node:crypto';
import { buildConnectionProfile } from '../config/fabric.js';
import { getIdentity } from './wallet.js';
import config from '../config/index.js';

// Cache gateway connections per user to avoid reconnecting on every request.
const gatewayCache = new Map();

function closeCached(cached) {
  cached.gateway.close();
  cached.grpcClient.close();
}

function newGrpcConnection(orgMSP) {
  const orgConfig = buildConnectionProfile(orgMSP);
  const tlsCert = fs.readFileSync(orgConfig.tlsCertPath);
  const tlsCredentials = grpc.credentials.createSsl(tlsCert);
  return new grpc.Client(
    `${orgConfig.peerHost}:${orgConfig.peerPort}`,
    tlsCredentials,
    { 'grpc.ssl_target_name_override': orgConfig.peerHost }
  );
}

function newIdentity(identity) {
  const certificate = identity.credentials.certificate;
  return { mspId: identity.mspId, credentials: Buffer.from(certificate) };
}

function newSigner(identity) {
  const privateKeyPem = identity.credentials.privateKey;
  const privateKey = crypto.createPrivateKey(privateKeyPem);
  return signers.newPrivateKeySigner(privateKey);
}

// Get or create a Gateway connection for a given user.
export async function getGateway(userId) {
  const now = Date.now();
  const cached = gatewayCache.get(userId);
  if (cached && now - cached.lastUsedAt < config.gatewayCache.ttlMs) {
    cached.lastUsedAt = now;
    return cached;
  }
  if (cached) {
    closeCached(cached);
    gatewayCache.delete(userId);
  }

  const identity = getIdentity(userId);
  if (!identity) throw Object.assign(new Error('身份未找到'), { code: 'UNAUTHORIZED' });

  const grpcClient = newGrpcConnection(identity.mspId);
  const gateway = connect({
    client: grpcClient,
    identity: newIdentity(identity),
    signer: newSigner(identity),
    evaluateOptions: () => ({ deadline: Date.now() + 5000 }),
    endorseOptions: () => ({ deadline: Date.now() + 15000 }),
    submitOptions: () => ({ deadline: Date.now() + 5000 }),
    commitStatusOptions: () => ({ deadline: Date.now() + 60000 }),
  });

  const entry = { gateway, grpcClient, lastUsedAt: now };
  gatewayCache.set(userId, entry);
  return entry;
}

// Get a contract handle for a given chaincode.
export async function getContract(userId, chaincodeName, contractName) {
  const { gateway } = await getGateway(userId);
  const network = gateway.getNetwork(config.fabric.channelName);
  return network.getContract(chaincodeName, contractName);
}

// Submit a transaction (read-write).
export async function submitTransaction(userId, chaincodeName, fn, ...args) {
  const contract = await getContract(userId, chaincodeName);
  const resultBytes = await contract.submitTransaction(fn, ...args);
  return resultBytes.length ? JSON.parse(new TextDecoder().decode(resultBytes)) : null;
}

// Evaluate a transaction (read-only query).
export async function evaluateTransaction(userId, chaincodeName, fn, ...args) {
  const contract = await getContract(userId, chaincodeName);
  const resultBytes = await contract.evaluateTransaction(fn, ...args);
  return resultBytes.length ? JSON.parse(new TextDecoder().decode(resultBytes)) : null;
}

export async function submitTransactionOptions(userId, chaincodeName, fn, options = {}) {
  const contract = await getContract(userId, chaincodeName, options.contractName);
  const resultBytes = await contract.submit(fn, {
    arguments: options.arguments || [],
    transientData: options.transientData,
    endorsingOrganizations: options.endorsingOrganizations,
  });
  return decodeResult(resultBytes);
}

export async function evaluateTransactionOptions(userId, chaincodeName, fn, options = {}) {
  const contract = await getContract(userId, chaincodeName, options.contractName);
  const resultBytes = await contract.evaluate(fn, {
    arguments: options.arguments || [],
    transientData: options.transientData,
    endorsingOrganizations: options.endorsingOrganizations,
  });
  return decodeResult(resultBytes);
}

function decodeResult(resultBytes) {
  if (!resultBytes?.length) return null;
  const text = new TextDecoder().decode(resultBytes);
  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

// Close a user's cached connection when they log out (optional).
export function closeGateway(userId) {
  const cached = gatewayCache.get(userId);
  if (cached) {
    closeCached(cached);
    gatewayCache.delete(userId);
  }
}

export function closeAllGateways() {
  for (const cached of gatewayCache.values()) closeCached(cached);
  gatewayCache.clear();
}

const cacheSweep = setInterval(() => {
  const now = Date.now();
  for (const [userId, cached] of gatewayCache) {
    if (now - cached.lastUsedAt >= config.gatewayCache.ttlMs) closeGateway(userId);
  }
}, Math.min(config.gatewayCache.ttlMs, 60_000));
cacheSweep.unref();
