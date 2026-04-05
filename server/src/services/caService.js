import FabricCAServices from 'fabric-ca-client';
import { buildConnectionProfile } from '../config/fabric.js';
import { putIdentity, getIdentity } from './wallet.js';

// Cache CA client instances per org
const caClients = {};

function getCAClient(orgMSP) {
  if (caClients[orgMSP]) return caClients[orgMSP];

  const orgConfig = buildConnectionProfile(orgMSP);
  const caUrl = `https://${orgConfig.caHost}:7054`;
  const caClient = new FabricCAServices(caUrl, {
    trustedRoots: [],
    verify: false, // dev only — accept self-signed CA certs
  }, orgConfig.caHost);

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
export async function registerAndEnrollUser(userId, orgMSP) {
  const caClient = getCAClient(orgMSP);

  // Ensure the CA admin is enrolled first
  const adminIdentity = await enrollAdmin(orgMSP);

  // Build an admin User object that fabric-ca-client can use for registration
  const adminUser = {
    getName: () => `admin-${orgMSP}`,
    getMSPId: () => orgMSP,
    getIdentity: () => ({
      serialize: () => adminIdentity.credentials.certificate,
    }),
    getSigningIdentity: () => ({
      sign: (msg) => {
        const { createSign } = require('node:crypto');
        const sign = createSign('SHA256');
        sign.update(msg);
        return sign.sign(adminIdentity.credentials.privateKey);
      },
    }),
  };

  // Register
  const secret = await caClient.register(
    {
      affiliation: '',
      enrollmentID: userId,
      role: 'client',
    },
    adminUser
  );

  // Enroll
  const enrollment = await caClient.enroll({
    enrollmentID: userId,
    enrollmentSecret: secret,
  });

  putIdentity(userId, orgMSP, enrollment.certificate, enrollment.key.toBytes());
  return getIdentity(userId);
}
