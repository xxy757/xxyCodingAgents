import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { terminalsApi } from '@/lib/api';
import type { TerminalSession } from '@/lib/types';

export function useTerminals() {
  return useQuery<TerminalSession[]>({ queryKey: ['terminals'], queryFn: terminalsApi.list });
}

export function useTerminal(id: string) {
  return useQuery<TerminalSession>({
    queryKey: ['terminals', id],
    queryFn: () => terminalsApi.get(id),
    enabled: !!id,
  });
}

export function useCreateTerminal() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: terminalsApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['terminals'] }),
  });
}
