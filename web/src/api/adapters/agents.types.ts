import type {
  AgentControlActionDTO,
  AgentControlResultDTO,
  AgentDetailDTO,
  AgentsListDTO,
} from "@/types/agents";

export type AgentsApiAdapter = {
  fetchList(): Promise<AgentsListDTO>;
  fetchDetail(id: string): Promise<AgentDetailDTO>;
  postControl(
    id: string,
    action: AgentControlActionDTO,
  ): Promise<AgentControlResultDTO>;
};
