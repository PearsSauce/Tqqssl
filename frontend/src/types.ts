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

export type DNSAccount = {
  id: string;
  name: string;
  provider: string;
  accessKeyMasked?: string;
  hasSecretKey: boolean;
  remark?: string;
  createdAt: string;
  updatedAt: string;
};

export type CertificateApplication = {
  id: string;
  primaryDomain: string;
  sans: string[];
  dnsAccountId: string;
  dnsAccountName?: string;
  challengeMode: "dns-01";
  status: "pending";
  createdAt: string;
  updatedAt: string;
};
