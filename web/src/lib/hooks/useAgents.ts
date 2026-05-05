import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { agentsApi } from '@/lib/api';
import type { AgentInstance } from '@/lib/types';

export function useAgents() {
  return useQuery<AgentInstance[]>({ queryKey: ['agents'], queryFn: agentsApi.list });
}

export function useAgentAction() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, action }: { id: string; action: 'pause' | 'resume' | 'stop' }) => {
      const actions = { pause: agentsApi.pause, resume: agentsApi.resume, stop: agentsApi.stop };
      return actions[action](id);
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }),
  });
}
