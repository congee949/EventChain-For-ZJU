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
      peerHost: 'peer0.platform.eventchain.com',
      peerPort: 7051,
      caHost: 'ca.platform.eventchain.com',
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/platform.eventchain.com/peers/peer0.platform.eventchain.com/tls/ca.crt'
      ),
    },
    OrganizerMSP: {
      mspId: 'OrganizerMSP',
      peerHost: 'peer0.organizer.eventchain.com',
      peerPort: 9051,
      caHost: 'ca.organizer.eventchain.com',
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/organizer.eventchain.com/peers/peer0.organizer.eventchain.com/tls/ca.crt'
      ),
    },
    StudentMSP: {
      mspId: 'StudentMSP',
      peerHost: 'peer0.student.eventchain.com',
      peerPort: 11051,
      caHost: 'ca.student.eventchain.com',
      tlsCertPath: path.join(
        networkRoot,
        'organizations/peerOrganizations/student.eventchain.com/peers/peer0.student.eventchain.com/tls/ca.crt'
      ),
    },
  };

  return orgMap[orgMSP] || orgMap.StudentMSP;
}
