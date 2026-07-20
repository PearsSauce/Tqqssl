import { Button, Card, Description, Form, Input, Label, Spinner, TextArea, TextField } from "@heroui/react";
import { useEffect, useMemo, useState } from "react";

import { apiRequest } from "../../../../../api";
import { errorMessage } from "../../../../../lib/utils";
import type { DNSAccount } from "../../../../../types";

import { EmptyState, InlineAlert } from "./shared";

export function DNSAccountsPanel({ accounts, onCreated, onUpdated, onDeleted }: {
  accounts: DNSAccount[];
  onCreated: (account: DNSAccount) => void;
  onUpdated: (account: DNSAccount) => void;
  onDeleted: (accountID: string) => void;
}) {
  const [name, setName] = useState("");
  const [provider, setProvider] = useState("alidns");
  const [accessKey, setAccessKey] = useState("");
  const [secretKey, setSecretKey] = useState("");
  const [remark, setRemark] = useState("");
  const [editingID, setEditingID] = useState("");
  const [editName, setEditName] = useState("");
  const [editProvider, setEditProvider] = useState("");
  const [editAccessKey, setEditAccessKey] = useState("");
  const [editSecretKey, setEditSecretKey] = useState("");
  const [editRemark, setEditRemark] = useState("");
  const [pending, setPending] = useState(false);
  const [updatingID, setUpdatingID] = useState("");
  const [deletingID, setDeletingID] = useState("");
  const [error, setError] = useState("");
  const editingAccount = useMemo(() => accounts.find((account) => account.id === editingID) || null, [accounts, editingID]);

  useEffect(() => {
    if (editingID && !editingAccount) {
      clearEditForm();
    }
  }, [editingAccount, editingID]);

  async function submit() {
    setPending(true);
    setError("");
    try {
      const account = await apiRequest<DNSAccount>("/dns-accounts", {
        method: "POST",
        body: { name, provider, accessKey, secretKey, remark }
      });
      onCreated(account);
      setName("");
      setAccessKey("");
      setSecretKey("");
      setRemark("");
    } catch (err) {
      setError(errorMessage(err, "创建 DNS 账号失败"));
    } finally {
      setPending(false);
    }
  }

  function startEdit(account: DNSAccount) {
    setEditingID(account.id);
    setEditName(account.name);
    setEditProvider(account.provider);
    setEditAccessKey("");
    setEditSecretKey("");
    setEditRemark(account.remark || "");
    setError("");
  }

  function clearEditForm() {
    setEditingID("");
    setEditName("");
    setEditProvider("");
    setEditAccessKey("");
    setEditSecretKey("");
    setEditRemark("");
  }

  async function submitUpdate() {
    if (!editingID) {
      return;
    }
    setUpdatingID(editingID);
    setError("");
    const body: Record<string, string> = {
      name: editName,
      provider: editProvider,
      remark: editRemark
    };
    if (editAccessKey.trim()) {
      body.accessKey = editAccessKey;
    }
    if (editSecretKey.trim()) {
      body.secretKey = editSecretKey;
    }
    try {
      const account = await apiRequest<DNSAccount>(`/dns-accounts/${editingID}`, {
        method: "PATCH",
        body
      });
      onUpdated(account);
      clearEditForm();
    } catch (err) {
      setError(errorMessage(err, "更新 DNS 账号失败"));
    } finally {
      setUpdatingID("");
    }
  }

  async function deleteAccount(accountID: string) {
    setDeletingID(accountID);
    setError("");
    try {
      await apiRequest<void>(`/dns-accounts/${accountID}`, { method: "DELETE" });
      if (editingID === accountID) {
        clearEditForm();
      }
      onDeleted(accountID);
    } catch (err) {
      setError(errorMessage(err, "删除 DNS 账号失败"));
    } finally {
      setDeletingID("");
    }
  }

  return (
    <Card className="gap-6 p-6">
      <Card.Header>
        <Card.Title>DNS 账号</Card.Title>
        <Card.Description>保存个人签发证书所需的 DNS API 凭据。SecretKey 会加密后写入本地数据文件，不会从 API 返回。</Card.Description>
      </Card.Header>
      <Card.Content className="grid gap-6">
        <Form
          className="grid gap-4"
          onSubmit={(event) => {
            event.preventDefault();
            void submit();
          }}
        >
          {error ? <InlineAlert status="danger" title={error} /> : null}
          <div className="grid gap-4 sm:grid-cols-2">
            <TextField isRequired fullWidth name="dnsName" value={name} onChange={setName}>
              <Label>账号名称</Label>
              <Input placeholder="阿里云主账号" />
            </TextField>
            <TextField isRequired fullWidth name="provider" value={provider} onChange={setProvider}>
              <Label>服务商标识</Label>
              <Input placeholder="alidns / dnspod / cloudflare" />
              <Description>仅使用小写字母、数字、下划线和短横线。</Description>
            </TextField>
          </div>
          <TextField fullWidth name="accessKey" value={accessKey} onChange={setAccessKey}>
            <Label>AccessKey</Label>
            <Input autoComplete="off" placeholder="可选，按服务商要求填写" />
          </TextField>
          <TextField isRequired fullWidth name="secretKey" type="password" value={secretKey} onChange={setSecretKey}>
            <Label>SecretKey / API Token</Label>
            <Input autoComplete="new-password" placeholder="创建后不会再展示明文" />
          </TextField>
          <div className="grid gap-2">
            <Label htmlFor="dns-remark">备注</Label>
            <TextArea
              fullWidth
              id="dns-remark"
              maxLength={300}
              placeholder="例如：只用于 example.com 的 DNS-01 验证"
              rows={3}
              value={remark}
              onChange={(event) => setRemark(event.target.value)}
            />
          </div>
          <Button isPending={pending} type="submit">
            {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}保存 DNS 账号</>}
          </Button>
        </Form>

        {editingAccount ? (
          <Form
            className="grid gap-4 rounded-3xl border border-blue-100 bg-blue-50/60 p-4"
            onSubmit={(event) => {
              event.preventDefault();
              void submitUpdate();
            }}
          >
            <div>
              <div className="font-medium text-slate-950">编辑 DNS 账号</div>
              <div className="mt-1 text-sm text-slate-500">AccessKey 和 SecretKey 留空表示保持原值；填写后会覆盖并加密保存。</div>
            </div>
            <div className="grid gap-4 sm:grid-cols-2">
              <TextField isRequired fullWidth name="editDnsName" value={editName} onChange={setEditName}>
                <Label>账号名称</Label>
                <Input placeholder="阿里云主账号" />
              </TextField>
              <TextField isRequired fullWidth name="editProvider" value={editProvider} onChange={setEditProvider}>
                <Label>服务商标识</Label>
                <Input placeholder="alidns / dnspod / cloudflare" />
              </TextField>
            </div>
            <TextField fullWidth name="editAccessKey" value={editAccessKey} onChange={setEditAccessKey}>
              <Label>新 AccessKey</Label>
              <Input autoComplete="off" placeholder={`留空保持当前值：${editingAccount.accessKeyMasked || "未填写"}`} />
            </TextField>
            <TextField fullWidth name="editSecretKey" type="password" value={editSecretKey} onChange={setEditSecretKey}>
              <Label>新 SecretKey / API Token</Label>
              <Input autoComplete="new-password" placeholder="留空保持当前密文，填写则轮换" />
            </TextField>
            <div className="grid gap-2">
              <Label htmlFor="edit-dns-remark">备注</Label>
              <TextArea
                fullWidth
                id="edit-dns-remark"
                maxLength={300}
                rows={3}
                value={editRemark}
                onChange={(event) => setEditRemark(event.target.value)}
              />
            </div>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Button isPending={updatingID === editingID} type="submit">
                {({ isPending }) => <>{isPending ? <Spinner color="current" size="sm" /> : null}保存修改</>}
              </Button>
              <Button type="button" variant="secondary" onPress={clearEditForm}>取消</Button>
            </div>
          </Form>
        ) : null}

        <div className="grid gap-3">
          {accounts.length === 0 ? (
            <EmptyState title="还没有 DNS 账号" description="先新增 DNS 账号，再创建证书申请。" />
          ) : (
            accounts.map((account) => (
              <div key={account.id} className="rounded-3xl border border-slate-200/80 bg-slate-50/80 p-4">
                <div className="flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between">
                  <div>
                    <div className="font-medium text-slate-950">{account.name}</div>
                    <div className="mt-1 text-sm text-slate-500">{account.provider} · {account.accessKeyMasked || "未填写 AccessKey"}</div>
                    {account.remark ? <div className="mt-2 text-sm leading-6 text-slate-500">{account.remark}</div> : null}
                  </div>
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Button variant="secondary" onPress={() => startEdit(account)}>编辑</Button>
                    <Button
                      variant="secondary"
                      isPending={deletingID === account.id}
                      onPress={() => void deleteAccount(account.id)}
                    >
                      删除
                    </Button>
                  </div>
                </div>
              </div>
            ))
          )}
        </div>
      </Card.Content>
    </Card>
  );
}
