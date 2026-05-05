import { useQuery } from '@tanstack/react-query';
import { systemApi } from '@/lib/api';
import type { ResourceSnapshot, HealthStatus, DiagnosticsData } from '@/lib/types';

export function useSystemMetrics(refetchInterval = 5000) {
  return useQuery<ResourceSnapshot>({
    queryKey: ['system', 'metrics'],
    queryFn: systemApi.metrics,
    refetchInterval,
  });
}

export function useDiagnostics() {
  return useQuery<DiagnosticsData>({
    queryKey: ['system', 'diagnostics'],
    queryFn: systemApi.diagnostics,
    refetchInterval: 10000,
  });
}

export function useHealth() {
  return useQuery<HealthStatus>({
    queryKey: ['health'],
    queryFn: systemApi.healthz,
    refetchInterval: 10000,
  });
}

export function useReady() {
  return useQuery<HealthStatus>({
    queryKey: ['ready'],
    queryFn: systemApi.readyz,
    refetchInterval: 10000,
  });
}
