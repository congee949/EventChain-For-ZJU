import FabricCAServices from 'fabric-ca-client';
import { User } from 'fabric-common';
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
// Idempotent: passes an explicit deterministic secret on first registration so
// re-enrollment works after server-side wallet/db wipes (the CA's identity DB
// outlives the server's SQLite, so a fresh server with a re-registered user
// would otherwise hit "Identity already registered" code 74 and have no way
// to re-enroll). The secret is derived from userId and is demo-only — for
// production, store a randomly generated secret in a secure key store.
export async function registerAndEnrollUser(userId, orgMSP) {
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

  // Deterministic per-user secret so re-enroll works after wallet wipe.
  const secret = `${userId}-pw`;

  // Register (idempotent — swallow "already registered" errors)
  try {
    await caClient.register(
      {
        affiliation: '',
        enrollmentID: userId,
        enrollmentSecret: secret,
        role: 'client',
      },
      adminUser
    );
  } catch (err) {
    const msg = String(err?.message || '');
    if (!msg.includes('already registered') && !msg.includes('code: 74')) {
      throw err;
    }
    // Already registered — proceed to enroll with the deterministic secret.
  }

  // Enroll
  const enrollment = await caClient.enroll({
    enrollmentID: userId,
    enrollmentSecret: secret,
  });

  putIdentity(userId, orgMSP, enrollment.certificate, enrollment.key.toBytes());
  return getIdentity(userId);
}
