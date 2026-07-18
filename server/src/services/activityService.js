import config from '../config/index.js';
import { evaluateTransactionOptions, submitTransactionOptions } from './fabricGateway.js';

const ENDORSERS = ['PlatformMSP', 'StudentMSP'];

export async function activitySubmit(userId, fn, args = [], transientData) {
  const transient = transientData
    ? Object.fromEntries(Object.entries(transientData).map(([key, value]) => [key, Buffer.from(JSON.stringify(value))]))
    : undefined;
  return submitTransactionOptions(userId, config.fabric.chaincode.activity, fn, {
    arguments: args.map(String), transientData: transient, endorsingOrganizations: ENDORSERS,
  });
}

export async function activityEvaluate(userId, fn, args = []) {
  return evaluateTransactionOptions(userId, config.fabric.chaincode.activity, fn, {
    arguments: args.map(String), endorsingOrganizations: ['PlatformMSP'],
  });
}
