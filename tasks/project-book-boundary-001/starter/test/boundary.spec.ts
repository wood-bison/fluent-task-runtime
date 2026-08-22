import { ownsBoundary } from '../src/boundary.ts';

export function learnerSmokeCheck(): boolean {
  return ownsBoundary({ owner: 'server', privateContent: false });
}
