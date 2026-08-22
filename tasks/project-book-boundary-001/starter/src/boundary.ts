export interface BoundaryInput {
  owner: 'server' | 'browser';
  privateContent: boolean;
}

export function ownsBoundary(input: BoundaryInput): boolean {
  return false;
}
