export type User = {
  id: string;
  email: string;
  name: string;
  public_key_pem?: string;
  created_at: string;
  updated_at: string;
};

export type LoginResponse = {
  success: true;
  data: {
    access_token: string;
    token_type: string;
    expires_at: string;
    user: User;
  };
};
