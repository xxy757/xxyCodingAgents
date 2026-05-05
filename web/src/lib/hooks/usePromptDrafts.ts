import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { promptDraftsApi } from '@/lib/api';
import type { PromptDraft } from '@/lib/types';

export function usePromptDrafts(projectId: string) {
  return useQuery<PromptDraft[]>({
    queryKey: ['prompt-drafts', projectId],
    queryFn: () => promptDraftsApi.list(projectId),
    enabled: !!projectId,
  });
}

export function useGeneratePromptDraft() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      projectId,
      input,
      taskType,
      techStackId,
    }: {
      projectId: string;
      input: string;
      taskType?: string;
      techStackId?: string;
    }) => promptDraftsApi.generate(projectId, input, taskType, techStackId),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: ['prompt-drafts', variables.projectId] }),
  });
}

export function useUpdatePromptDraft() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      id,
      finalPrompt,
      taskType,
    }: {
      id: string;
      finalPrompt: string;
      taskType?: string;
    }) => promptDraftsApi.update(id, finalPrompt, taskType),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['prompt-drafts'] }),
  });
}

export function useSendPromptDraft() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => promptDraftsApi.send(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['prompt-drafts'] }),
  });
}
