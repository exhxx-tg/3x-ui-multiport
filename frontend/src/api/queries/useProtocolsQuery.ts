import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { useMemo } from 'react';
import { HttpUtil } from '@/utils';
import { parseMsg } from '@/utils/zodValidate';
import { keys } from '@/api/queryKeys';
import { ProtocolDetailedSchema } from '@/schemas/protocols/protocol-ecosystem';
import type { ProtocolInfo } from '@/models/protocol-ecosystem';

async function fetchProtocolsDetailed(): Promise<ProtocolInfo[]> {
  const msg = await HttpUtil.get('/panel/api/protocols/detailed', undefined, { silent: true });
  if (!msg?.success) throw new Error(msg?.msg || 'Failed to fetch protocols');
  const validated = parseMsg(msg, ProtocolDetailedSchema, 'protocols/detailed');
  return Array.isArray(validated.obj) ? validated.obj : [];
}

async function startProtocol(id: string) {
  const msg = await HttpUtil.post(`/panel/api/protocols/${id}/start`);
  if (!msg?.success) throw new Error(msg?.msg || `Failed to start ${id}`);
  return msg;
}

async function stopProtocol(id: string) {
  const msg = await HttpUtil.post(`/panel/api/protocols/${id}/stop`);
  if (!msg?.success) throw new Error(msg?.msg || `Failed to stop ${id}`);
  return msg;
}

async function restartProtocol(id: string) {
  const msg = await HttpUtil.post(`/panel/api/protocols/${id}/restart`);
  if (!msg?.success) throw new Error(msg?.msg || `Failed to restart ${id}`);
  return msg;
}

export function useProtocolsQuery() {
  const query = useQuery({
    queryKey: [...keys.server.status(), 'protocols-detailed'],
    queryFn: fetchProtocolsDetailed,
    refetchInterval: 5000,
    staleTime: 0,
  });

  const protocols = useMemo(() => query.data ?? [], [query.data]);

  return {
    protocols,
    fetched: query.data !== undefined || query.isError,
    fetchError: query.error ? (query.error as Error).message : '',
    isLoading: query.isLoading,
    refresh: async () => { await query.refetch(); },
  };
}

export function useProtocolMutations() {
  const queryClient = useQueryClient();

  const startMutation = useMutation({
    mutationFn: startProtocol,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...keys.server.status(), 'protocols-detailed'] });
    },
  });

  const stopMutation = useMutation({
    mutationFn: stopProtocol,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...keys.server.status(), 'protocols-detailed'] });
    },
  });

  const restartMutation = useMutation({
    mutationFn: restartProtocol,
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: [...keys.server.status(), 'protocols-detailed'] });
    },
  });

  return {
    startProtocol: startMutation.mutateAsync,
    stopProtocol: stopMutation.mutateAsync,
    restartProtocol: restartMutation.mutateAsync,
    isStarting: startMutation.isPending,
    isStopping: stopMutation.isPending,
    isRestarting: restartMutation.isPending,
  };
}

export async function fetchProtocolHealth(id: string): Promise<{ healthy: boolean; error: string }> {
  const msg = await HttpUtil.get<{ healthy?: boolean; error?: string }>(`/panel/api/protocols/${id}/health`);
  if (!msg?.success) return { healthy: false, error: (msg?.msg as string) || 'Unknown error' };
  return { healthy: msg.obj?.healthy ?? false, error: msg.obj?.error ?? '' };
}
