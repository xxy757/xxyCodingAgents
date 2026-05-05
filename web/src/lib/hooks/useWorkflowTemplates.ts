import { useQuery } from '@tanstack/react-query';
import { workflowTemplatesApi } from '@/lib/api';
import type { WorkflowTemplate } from '@/lib/types';

export function useWorkflowTemplates() {
  return useQuery<WorkflowTemplate[]>({
    queryKey: ['workflow-templates'],
    queryFn: workflowTemplatesApi.list,
  });
}
