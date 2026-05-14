export type LoginInput = {
  account: string;
  password: string;
};

export type LoginResultDTO = {
  ok: boolean;
  accessToken?: string;
  expiresAt?: number;
};
