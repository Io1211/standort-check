import { useSyncExternalStore } from 'react'
import { getAdminRevision, subscribeAdminUpdates } from '../api'

export function useAdminRevision() {
  return useSyncExternalStore(subscribeAdminUpdates, getAdminRevision, getAdminRevision)
}
