/** Backend/session transport shape — mirror SaaS contracts here when available. */
export type AuthSessionDTO = {
  user: AuthUserDTO | null;
};

export type AuthUserDTO = {
  id: string;
  email: string;
  displayName?: string;
};
