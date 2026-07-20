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
  status: "pending" | "ordered";
  orderUrl?: string;
  orderStatus?: string;
  authorizationUrls?: string[];
  finalizeUrl?: string;
  privateKeyReady: boolean;
  csrReady: boolean;
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

export type CertificateAuthorization = {
  url: string;
  domain: string;
  wildcard: boolean;
  status: string;
  expires?: string;
  dns01?: DNS01Challenge;
};

export type DNS01Challenge = {
  url: string;
  status: string;
  token: string;
  keyAuthorization: string;
  recordName: string;
  recordType: "TXT";
  recordValue: string;
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
