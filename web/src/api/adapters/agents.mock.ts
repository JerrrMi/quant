import type {
  AgentControlActionDTO,
  AgentControlResultDTO,
  AgentDetailDTO,
  AgentsListDTO,
} from "@/types/agents";

import {
  buildAgentDetailDTO,
  buildAgentsListDTO,
} from "@/lib/console-seed-agents-accounts";

import type { AgentsApiAdapter } from "./agents.types";

function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

async function fetchList(): Promise<AgentsListDTO> {
  await delay(240);
  return buildAgentsListDTO();
}

async function fetchDetail(id: string): Promise<AgentDetailDTO> {
  await delay(260);
  const d = buildAgentDetailDTO(id);
  if (!d) {
    throw new Error(`Agent not found: ${id}`);
  }
  return d;
}

async function postControl(
  id: string,
  action: AgentControlActionDTO,
): Promise<AgentControlResultDTO> {
  await delay(400);
  const ok = id !== "agent-local-spot" || action === "refresh";
  return {
    ok,
    message: ok
      ? `Mock：已对 ${id} 执行「${action}」`
      : `Mock：离线 Agent 拒绝「${action}」，请先 start`,
    appliedAt: new Date().toISOString(),
  };
}

export const agentsMockAdapter: AgentsApiAdapter = {
  fetchList,
  fetchDetail,
  postControl,
};
