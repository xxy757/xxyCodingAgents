import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { gatesApi } from '@/lib/api';
import type { Gate } from '@/lib/types';

export function useGates(runId: string) {
  return useQuery<Gate[]>({
    queryKey: ['gates', runId],
    queryFn: () => gatesApi.list(runId),
    enabled: !!runId,
  });
}

export function useApproveGate(runId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ gateId, approvedBy }: { gateId: string; approvedBy?: string }) =>
      gatesApi.approve(gateId, approvedBy),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['gates', runId] });
      qc.invalidateQueries({ queryKey: ['runs', runId, 'workflow'] });
    },
  });
}
