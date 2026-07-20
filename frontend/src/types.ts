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

export type CertificatePrecheck = {
  primaryDomain: string;
  sans: string[];
  dnsAccountId: string;
  dnsAccountName: string;
  dnsProvider: string;
  challengeMode: "dns-01";
  domainCount: number;
  warnings: string[];
};

export type ACMEStatus = {
  accountKeyReady: boolean;
  accountKeyType?: string;
  directoryUrl?: string;
  termsAgreed: boolean;
  ready: boolean;
  accountRegistered: boolean;
  accountUrl?: string;
  accountStatus?: string;
  contactEmail?: string;
};

export type ACMEDirectoryCheck = {
  directoryUrl: string;
  newNonce: string;
  newAccount: string;
  newOrder: string;
  termsOfService?: string;
  website?: string;
  externalAccountRequired: boolean;
  warnings: string[];
};

export type ACMEAccountRegistration = {
  accountRegistered: boolean;
  accountUrl: string;
  accountStatus: string;
  contactEmail: string;
};
