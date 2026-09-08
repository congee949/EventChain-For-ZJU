import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Connection profile for the Fabric Gateway SDK.
// In production the TLS cert paths come from the crypto material generated
// by the Fabric CA / cryptogen tool.  During development they live under the
// network's organizations/ directory.

const networkRoot = path.resolve(__dirname, '../../../fabric/network');

function port(name, fallback) {
  const value = Number.parseInt(process.env[name], 10);
  return Number.isSafeInteger(value) && value > 0 && value <= 65535 ? value : fallback;
}

export function buildConnectionProfile(orgMSP) {
  const orgMap = {
    PlatformMSP: {
      mspId: 'PlatformMSP',
      peerHost: process.env.FABRIC_PLATFORM_PEER_HOST || 'localhost',
      peerPort: port('FABRIC_PLATFORM_PEER_PORT', 7051),
      caHost: process.env.FABRIC_PLATFORM_CA_HOST || 'localhost',
      caPort: port('FABRIC_PLATFORM_CA_PORT', 7054),
      caTlsCertPath: path.join(networkRoot, 'organizations/fabric-ca/platform/ca-cert.pem'),
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/platform.eventchain.com/peers/peer0.platform.eventchain.com/tls/ca.crt'
      ),
    },
    OrganizerMSP: {
      mspId: 'OrganizerMSP',
      peerHost: process.env.FABRIC_ORGANIZER_PEER_HOST || 'localhost',
      peerPort: port('FABRIC_ORGANIZER_PEER_PORT', 9051),
      caHost: process.env.FABRIC_ORGANIZER_CA_HOST || 'localhost',
      caPort: port('FABRIC_ORGANIZER_CA_PORT', 8054),
      caTlsCertPath: path.join(networkRoot, 'organizations/fabric-ca/organizer/ca-cert.pem'),
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/organizer.eventchain.com/peers/peer0.organizer.eventchain.com/tls/ca.crt'
      ),
    },
    StudentMSP: {
      mspId: 'StudentMSP',
      peerHost: process.env.FABRIC_STUDENT_PEER_HOST || 'localhost',
      peerPort: port('FABRIC_STUDENT_PEER_PORT', 11051),
      caHost: process.env.FABRIC_STUDENT_CA_HOST || 'localhost',
      caPort: port('FABRIC_STUDENT_CA_PORT', 9054),
      caTlsCertPath: path.join(networkRoot, 'organizations/fabric-ca/student/ca-cert.pem'),
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/student.eventchain.com/peers/peer0.student.eventchain.com/tls/ca.crt'
      ),
    },
  };

  const profile = orgMap[orgMSP];
  if (!profile) throw new Error(`unsupported Fabric MSP: ${orgMSP}`);
  return profile;
}
