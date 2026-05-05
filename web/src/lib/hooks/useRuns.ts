import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { runsApi } from '@/lib/api';
import type { Run, Task, Event, WorkflowGraphData } from '@/lib/types';

export function useRuns() {
  return useQuery<Run[]>({ queryKey: ['runs'], queryFn: runsApi.list });
}

export function useRun(id: string) {
  return useQuery<Run>({ queryKey: ['runs', id], queryFn: () => runsApi.get(id), enabled: !!id });
}

export function useRunTasks(id: string) {
  return useQuery<Task[]>({ queryKey: ['runs', id, 'tasks'], queryFn: () => runsApi.getTasks(id), enabled: !!id });
}

export function useRunTimeline(id: string) {
  return useQuery<Event[]>({ queryKey: ['runs', id, 'timeline'], queryFn: () => runsApi.getTimeline(id), enabled: !!id });
}

export function useRunWorkflow(id: string) {
  return useQuery<WorkflowGraphData>({ queryKey: ['runs', id, 'workflow'], queryFn: () => runsApi.getWorkflow(id), enabled: !!id });
}

export function useCreateRun() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: runsApi.create,
    onSuccess: () => qc.invalidateQueries({ queryKey: ['runs'] }),
  });
}
