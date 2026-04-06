import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

// Connection profile for the Fabric Gateway SDK.
// In production the TLS cert paths come from the crypto material generated
// by the Fabric CA / cryptogen tool.  During development they live under the
// network's organizations/ directory.

const networkRoot = path.resolve(__dirname, '../../../fabric/network');

export function buildConnectionProfile(orgMSP) {
  const orgMap = {
    PlatformMSP: {
      mspId: 'PlatformMSP',
      peerHost: 'localhost',
      peerPort: 7051,
      caHost: 'localhost',
      caPort: 7054,
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/platform.eventchain.com/peers/peer0.platform.eventchain.com/tls/ca.crt'
      ),
    },
    OrganizerMSP: {
      mspId: 'OrganizerMSP',
      peerHost: 'localhost',
      peerPort: 9051,
      caHost: 'localhost',
      caPort: 8054,
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/organizer.eventchain.com/peers/peer0.organizer.eventchain.com/tls/ca.crt'
      ),
    },
    StudentMSP: {
      mspId: 'StudentMSP',
      peerHost: 'localhost',
      peerPort: 11051,
      caHost: 'localhost',
      caPort: 9054,
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/student.eventchain.com/peers/peer0.student.eventchain.com/tls/ca.crt'
      ),
    },
  };

  return orgMap[orgMSP] || orgMap.StudentMSP;
}
