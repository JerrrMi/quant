/** 持仓方向（做空策略常用 short）。 */
export type PositionSideKind = "long" | "short" | "flat";

export type AccountPositionDTO = {
  symbol: string;
  side: PositionSideKind;
  /** 合约：张/币本位由后端约定；前端按数值展示 */
  positionSize: number;
  leverage: number;
  /** 0–1+，可大于 1 表示高风险 */
  riskRatio: number;
  unrealizedPnl: number;
  marginUsed: number;
};

export type AccountVenueSliceDTO = {
  label: string;
  totalEquity: number;
  availableBalance: number;
  marginUsed: number;
  unrealizedPnl: number;
  realizedPnl: number;
  /** 聚合持仓方向（净敞口）。 */
  netPositionSide: PositionSideKind;
  /** 名义仓位规模汇总 */
  netPositionSize: number;
  /** 加权或账户级杠杆展示 */
  leverage: number;
  riskRatio: number;
  positions: AccountPositionDTO[];
};

export type AccountsOverviewDTO = {
  updatedAt: string;
  spot: AccountVenueSliceDTO;
  futures: AccountVenueSliceDTO;
};

export type AccountHistoryPointDTO = {
  at: string;
  /** 与 metric 对应的标量，如 USDT 权益 */
  value: number;
};

export type AccountsHistoryDTO = {
  venue: "spot" | "futures";
  metric: "equity" | "available";
  points: AccountHistoryPointDTO[];
  generatedAt: string;
};
