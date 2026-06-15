const feishuH5SDKURL = "https://lf-scm-cn.feishucdn.com/lark/op/h5-js-sdk-1.5.30.js";
const feishuOAuthAuthorizeURL = "https://open.feishu.cn/open-apis/authen/v1/authorize";

type FeishuWindow = Window & {
  h5sdk?: any;
  tt?: any;
  lark?: any;
};

type AuthCodeAttempt = {
  method: any;
  params: Record<string, unknown>;
};

let sdkLoadPromise: Promise<void> | null = null;

function feishuWindow(): FeishuWindow {
  return window as FeishuWindow;
}

function normalizeFeishuError(error: unknown, fallback: string): Error {
  if (error instanceof Error) return error;
  if (typeof error === "string" && error.trim()) return new Error(error.trim());
  if (error && typeof error === "object") {
    const record = error as Record<string, unknown>;
    for (const value of [record.msg, record.message, record.errMsg]) {
      if (typeof value === "string" && value.trim()) {
        return new Error(value.trim());
      }
    }
  }
  return new Error(fallback);
}

function sleep(ms: number) {
  return new Promise((resolve) => {
    window.setTimeout(resolve, ms);
  });
}

function extractAuthCode(payload: unknown): string {
  if (typeof payload === "string") return payload.trim();
  if (!payload || typeof payload !== "object") return "";

  const record = payload as Record<string, unknown>;
  for (const value of [record.code, record.authCode, record.tmpCode, record.accessCode]) {
    if (typeof value === "string" && value.trim()) {
      return value.trim();
    }
  }

  if (record.data && typeof record.data === "object") {
    return extractAuthCode(record.data);
  }
  return "";
}

function invokeSDKMethod(method: any, params: Record<string, unknown>): Promise<unknown> {
  if (typeof method !== "function") {
    return Promise.reject(new Error("method unavailable"));
  }

  return new Promise((resolve, reject) => {
    let settled = false;
    const finish = (fn: () => void) => {
      if (settled) return;
      settled = true;
      fn();
    };
    const onSuccess = (value: unknown) => finish(() => resolve(value));
    const onFail = (error: unknown) =>
      finish(() => reject(normalizeFeishuError(error, "飞书授权失败")));

    try {
      const result = method({
        ...params,
        success: onSuccess,
        fail: onFail,
        onSuccess,
        onFail,
      });
      if (result && typeof result.then === "function") {
        result.then(onSuccess).catch(onFail);
        return;
      }
      if (typeof result === "string") {
        onSuccess(result);
      }
    } catch (error) {
      onFail(error);
    }
  });
}

function authCodeAttempts(win: FeishuWindow, appID: string): AuthCodeAttempt[] {
  return [
    { method: win.tt?.requestAuthCode, params: { appId: appID, appID } },
    { method: win.tt?.requestAuthCode, params: { appId: appID } },
    { method: win.tt?.requestAuthCode, params: { appID } },
    { method: win.tt?.requestAuthCode, params: {} },
    { method: win.lark?.requestAuthCode, params: { appId: appID, appID } },
    { method: win.lark?.requestAuthCode, params: { appId: appID } },
    { method: win.h5sdk?.requestAuthCode, params: { appId: appID, appID } },
    { method: win.h5sdk?.requestAuthCode, params: { appId: appID } },
    { method: win.h5sdk?.requestAuthCode, params: {} },
  ].filter((attempt) => typeof attempt.method === "function");
}

async function waitForFeishuBridge(timeoutMs = 5000): Promise<FeishuWindow> {
  const startedAt = Date.now();
  while (Date.now() - startedAt < timeoutMs) {
    const win = feishuWindow();
    if (
      typeof win.tt?.requestAuthCode === "function" ||
      typeof win.lark?.requestAuthCode === "function" ||
      typeof win.h5sdk?.requestAuthCode === "function"
    ) {
      return win;
    }
    await sleep(120);
  }
  return feishuWindow();
}

async function tryConfigFeishuSDK(appID: string): Promise<void> {
  const sdk = feishuWindow().h5sdk || feishuWindow().lark;
  if (!sdk || typeof sdk.config !== "function") return;

  await new Promise<void>((resolve, reject) => {
    let settled = false;
    const finish = (fn: () => void) => {
      if (settled) return;
      settled = true;
      fn();
    };
    const timer = window.setTimeout(() => finish(resolve), 1500);
    const success = () => {
      window.clearTimeout(timer);
      finish(resolve);
    };
    const failure = (error: unknown) => {
      window.clearTimeout(timer);
      finish(() => reject(normalizeFeishuError(error, "飞书 SDK 初始化失败")));
    };

    try {
      if (typeof sdk.ready === "function") {
        sdk.ready(success);
      }
      if (typeof sdk.error === "function") {
        sdk.error(failure);
      }
      sdk.config({ appId: appID });
    } catch (error) {
      failure(error);
    }
  }).catch(() => undefined);
}

export function isFeishuClient(): boolean {
  if (typeof window === "undefined") return false;
  const ua = window.navigator.userAgent.toLowerCase();
  return ua.includes("feishu") || ua.includes("lark");
}

export function loadFeishuSDK() {
  if (typeof window === "undefined") return Promise.resolve();
  if (document.querySelector(`script[src="${feishuH5SDKURL}"]`)) {
    return Promise.resolve();
  }
  if (sdkLoadPromise) return sdkLoadPromise;

  sdkLoadPromise = new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = feishuH5SDKURL;
    script.async = true;
    script.defer = true;
    script.crossOrigin = "anonymous";
    script.onload = () => resolve();
    script.onerror = () => reject(new Error("飞书 SDK 加载失败"));
    document.body.append(script);
  });
  return sdkLoadPromise;
}

export async function getFeishuAuthCode(appID: string): Promise<string> {
  await loadFeishuSDK().catch(() => undefined);
  await waitForFeishuBridge();
  await tryConfigFeishuSDK(appID);
  const win = await waitForFeishuBridge(1200);
  const candidates = authCodeAttempts(win, appID);
  if (candidates.length === 0) {
    throw new Error("飞书授权方法不可用，请确认在飞书客户端内打开，并已配置网页应用可信域名");
  }

  let lastError: unknown = null;
  for (const candidate of candidates) {
    try {
      const result = await invokeSDKMethod(candidate.method, candidate.params);
      const code = extractAuthCode(result);
      if (code) return code;
    } catch (error) {
      lastError = error;
    }
  }
  throw normalizeFeishuError(lastError, "当前飞书容器未返回授权码");
}

export function buildFeishuOAuthURL(appID: string, redirectURI: string, state: string): string {
  const params = new URLSearchParams({
    app_id: appID,
    redirect_uri: redirectURI,
    response_type: "code",
    state,
  });
  return `${feishuOAuthAuthorizeURL}?${params.toString()}`;
}
