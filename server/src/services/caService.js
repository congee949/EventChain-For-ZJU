import FabricCAServices from 'fabric-ca-client';
import { User } from 'fabric-common';
import crypto from 'node:crypto';
import { buildConnectionProfile } from '../config/fabric.js';
import { putIdentity, getIdentity } from './wallet.js';

// Cache CA client instances per org
const caClients = {};

function getCAClient(orgMSP) {
  if (caClients[orgMSP]) return caClients[orgMSP];

  const orgConfig = buildConnectionProfile(orgMSP);
  const caUrl = `https://${orgConfig.caHost}:${orgConfig.caPort}`;
  const caClient = new FabricCAServices(caUrl, {
    trustedRoots: [],
    verify: false, // dev only — accept self-signed CA certs
  });

  caClients[orgMSP] = caClient;
  return caClient;
}

// Enroll the bootstrap admin identity for a given org CA.
// This admin identity is used to register new users.
export async function enrollAdmin(orgMSP) {
  if (getIdentity(`admin-${orgMSP}`)) return getIdentity(`admin-${orgMSP}`);

  const caClient = getCAClient(orgMSP);
  const enrollment = await caClient.enroll({
    enrollmentID: 'admin',
    enrollmentSecret: 'adminpw',
  });

  putIdentity(`admin-${orgMSP}`, orgMSP, enrollment.certificate, enrollment.key.toBytes());
  return getIdentity(`admin-${orgMSP}`);
}

// Register and enroll a new user with the org's CA.
//
// V2 uses a random opaque enrollment ID and random one-time enrollment secret.
// The demo reset removes both the CA database and the local wallet together.
export async function registerAndEnrollUser(accountId, orgMSP, role) {
  const caClient = getCAClient(orgMSP);

  // Ensure the CA admin is enrolled first
  const adminIdentity = await enrollAdmin(orgMSP);

  // Build admin User object required by fabric-ca-client register()
  const adminUser = User.createUser(
    `admin-${orgMSP}`,
    '',
    orgMSP,
    adminIdentity.credentials.certificate,
    adminIdentity.credentials.privateKey
  );

  const secret = crypto.randomBytes(24).toString('hex');

  await caClient.register(
    {
        affiliation: '',
        enrollmentID: accountId,
        enrollmentSecret: secret,
        role: 'client',
		attrs: [
		  { name: 'eventchain.accountID', value: accountId, ecert: true },
		  { name: 'eventchain.role', value: role, ecert: true },
		],
    },
    adminUser
  );

  // Enroll
  const enrollment = await caClient.enroll({
    enrollmentID: accountId,
    enrollmentSecret: secret,
  });

  putIdentity(accountId, orgMSP, enrollment.certificate, enrollment.key.toBytes());
  return getIdentity(accountId);
}
