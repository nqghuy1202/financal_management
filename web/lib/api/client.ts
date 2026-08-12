import type { Envelope, Session } from "./types";

const BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const API = `${BASE_URL}/api/v1`;

/**
 * Mã lỗi nghiệp vụ của backend, xem internal/pkg/response/code.go.
 * Chỉ khai báo những mã frontend thực sự phản ứng khác nhau.
 */
export const CODE = {
  TOKEN_MISSING: 41001,
  TOKEN_INVALID: 41002,
  TOKEN_EXPIRED: 41003,
  CREDENTIALS_INVALID: 41004,
  NOT_FOUND: 44000,
  TOO_MANY_REQUESTS: 42900,
} as const;

/**
 * Lỗi mang theo mã nghiệp vụ và thông báo của backend.
 *
 * Backend đã trả thông báo bằng tiếng Việt cho người dùng cuối, nên UI
 * hiển thị thẳng `message` thay vì tự dịch lại mã — một chỗ sinh chữ,
 * không phải hai.
 */
export class ApiError extends Error {
  constructor(
    readonly code: number,
    message: string,
    readonly status: number,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

// ---------------------------------------------------------------------
// Access token: chỉ nằm trong bộ nhớ
// ---------------------------------------------------------------------

/**
 * Access token cố tình KHÔNG ghi vào localStorage hay cookie đọc được.
 *
 * localStorage đọc được bằng JavaScript, nên một lỗ XSS là mất token.
 * Giữ trong biến module thì token biến mất khi tải lại trang — và đó là
 * lý do có `restoreSession()`: lúc khởi động app ta đổi cookie refresh
 * (HttpOnly, trình duyệt tự gửi) lấy một access token mới.
 */
let accessToken: string | null = null;

export function setAccessToken(token: string | null) {
  accessToken = token;
}

export function getAccessToken() {
  return accessToken;
}

/** Được AuthProvider gán, để client báo ngược lên khi phiên hết hạn hẳn. */
let onSessionLost: (() => void) | null = null;

export function setSessionLostHandler(fn: (() => void) | null) {
  onSessionLost = fn;
}

// ---------------------------------------------------------------------
// Gọi API
// ---------------------------------------------------------------------

type RequestOptions = {
  method?: string;
  body?: unknown;
  query?: Record<string, string | number | null | undefined>;
  /** Bỏ qua bước tự refresh — dùng cho chính các endpoint auth. */
  skipRefresh?: boolean;
};

function buildURL(path: string, query?: RequestOptions["query"]) {
  const url = new URL(API + path);
  for (const [key, value] of Object.entries(query ?? {})) {
    // Bỏ qua giá trị rỗng để không gửi `?type=` khiến backend hiểu nhầm
    // là có lọc.
    if (value === null || value === undefined || value === "") continue;
    url.searchParams.set(key, String(value));
  }
  return url.toString();
}

async function rawFetch(path: string, options: RequestOptions): Promise<Response> {
  const headers: Record<string, string> = {};
  if (options.body !== undefined) headers["Content-Type"] = "application/json";
  if (accessToken) headers["Authorization"] = `Bearer ${accessToken}`;

  return fetch(buildURL(path, options.query), {
    method: options.method ?? "GET",
    headers,
    body: options.body === undefined ? undefined : JSON.stringify(options.body),
    // Bắt buộc: cookie refresh do API đặt chỉ được gửi kèm khi có dòng này,
    // vì frontend (cổng 3000) và API (cổng 8080) là hai origin khác nhau.
    credentials: "include",
  });
}

async function parse<T>(res: Response): Promise<T> {
  let envelope: Envelope<T>;
  try {
    envelope = (await res.json()) as Envelope<T>;
  } catch {
    // Backend luôn trả JSON; tới đây nghĩa là API chết hoặc bị proxy chặn.
    throw new ApiError(0, "Không kết nối được máy chủ", res.status);
  }

  if (!res.ok) {
    throw new ApiError(envelope.code, envelope.message, res.status);
  }
  return envelope.data as T;
}

/**
 * Gộp mọi lần refresh đang chạy vào chung một promise.
 *
 * Khi mở dashboard, năm request cùng lúc đều nhận 401. Nếu mỗi cái tự gọi
 * /auth/refresh thì cái đầu tiên đổi được token, bốn cái sau gửi lên token
 * đã bị xoá (backend dùng GETDEL nên refresh token chỉ dùng được một lần)
 * và người dùng bị đá ra ngoài dù vừa đăng nhập xong.
 */
let refreshing: Promise<Session> | null = null;

function refreshSession(): Promise<Session> {
  refreshing ??= rawFetch("/auth/refresh", { method: "POST" })
    .then((res) => parse<Session>(res))
    .then((session) => {
      setAccessToken(session.access_token);
      return session;
    })
    .finally(() => {
      refreshing = null;
    });
  return refreshing;
}

/** Ba mã này đều có nghĩa "access token không dùng được nữa". */
function isTokenProblem(err: unknown) {
  return (
    err instanceof ApiError &&
    (err.code === CODE.TOKEN_EXPIRED ||
      err.code === CODE.TOKEN_INVALID ||
      err.code === CODE.TOKEN_MISSING)
  );
}

export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const res = await rawFetch(path, options);

  if (res.status !== 401 || options.skipRefresh) {
    return parse<T>(res);
  }

  // 401: thử đổi cookie refresh lấy access token mới rồi gọi lại đúng một
  // lần. Không lặp vô hạn — lần thứ hai mà vẫn 401 thì phiên đã hỏng thật.
  try {
    await refreshSession();
  } catch (err) {
    setAccessToken(null);
    onSessionLost?.();
    throw isTokenProblem(err)
      ? new ApiError(CODE.TOKEN_INVALID, "Phiên đăng nhập đã hết hạn, vui lòng đăng nhập lại", 401)
      : err;
  }

  return parse<T>(await rawFetch(path, options));
}

/**
 * Khôi phục phiên khi tải lại trang.
 *
 * Trả về null (không ném lỗi) khi chưa từng đăng nhập: người mở trang lần
 * đầu không có cookie, và đó là trạng thái bình thường chứ không phải sự cố.
 */
export async function restoreSession(): Promise<Session | null> {
  try {
    return await refreshSession();
  } catch {
    setAccessToken(null);
    return null;
  }
}
