export type User = {
  id: string;
  username: string;
  email: string;
  role: "admin";
  createdAt: string;
  updatedAt: string;
  lastLoginAt?: string;
};

export type RegisterOptions = {
  allowRegister: boolean;
};
