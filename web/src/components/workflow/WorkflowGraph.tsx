'use client';

import { useCallback, useMemo } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  useNodesState,
  useEdgesState,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import type { WorkflowGraphData } from '@/lib/types';
import { TaskNode } from './TaskNode';
import { GateNode } from './GateNode';

const nodeTypes = { task: TaskNode, gate: GateNode };

interface WorkflowGraphProps {
  graph: WorkflowGraphData | null;
  onApproveGate?: (gateId: string) => void;
}

export function WorkflowGraph({ graph, onApproveGate }: WorkflowGraphProps) {
  const initialNodes = useMemo(
    () =>
      (graph?.nodes || []).map((n) => ({
        ...n,
        data: { ...n.data, onApprove: onApproveGate },
      })),
    [graph, onApproveGate],
  );

  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(graph?.edges || []);

  // 当 graph 变化时更新 nodes/edges
  useMemo(() => {
    if (graph) {
      setNodes(graph.nodes.map((n) => ({ ...n, data: { ...n.data, onApprove: onApproveGate } })));
      setEdges(graph.edges);
    }
  }, [graph, onApproveGate, setNodes, setEdges]);

  if (!graph || graph.nodes.length === 0) {
    return (
      <div className="flex items-center justify-center h-64 text-sm text-[rgba(0,0,0,0.45)]">
        暂无工作流数据
      </div>
    );
  }

  return (
    <div className="h-[500px] bg-white rounded-lg border border-[#f0f0f0]">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        onNodesChange={onNodesChange}
        onEdgesChange={onEdgesChange}
        nodeTypes={nodeTypes}
        fitView
        defaultEdgeOptions={{ animated: true, style: { stroke: '#bfbfbf' } }}
      >
        <Background gap={16} size={1} color="#f0f0f0" />
        <Controls />
        <MiniMap
          nodeColor={(node) => {
            const status = node.data?.status;
            if (status === 'completed') return '#52c41a';
            if (status === 'running') return '#1677ff';
            if (status === 'failed') return '#ff4d4f';
            return '#d9d9d9';
          }}
          maskColor="rgba(0,0,0,0.05)"
        />
      </ReactFlow>
    </div>
  );
}
