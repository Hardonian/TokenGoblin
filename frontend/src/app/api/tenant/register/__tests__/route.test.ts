import { POST } from "../route";

jest.mock("next/server", () => ({
  NextResponse: {
    json: jest.fn((body, init) => ({
      status: init?.status ?? 200,
      json: async () => body,
      cookies: { set: jest.fn() },
    })),
  },
}));

const mockFetch = jest.fn();
global.fetch = mockFetch;

function request(body: unknown): Request {
  return {
    json: async () => body,
  } as unknown as Request;
}

describe("POST /api/tenant/register", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("rejects a missing organization name", async () => {
    const response = await POST(request({}));
    expect(response.status).toBe(400);
    await expect(response.json()).resolves.toMatchObject({
      ok: false,
      error: { code: "invalid_request" },
    });
  });

  it("accepts registration without a caller-selected tenant id", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({
        ok: true,
        data: { tenant_id: "generated-id", api_key: "key_1.tg_secret" },
      }),
    });

    const response = await POST(request({ name: "Acme" }));
    expect(response.status).toBe(200);
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/tenant/register"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ tenant_id: "", name: "Acme" }),
      }),
    );
    expect(response.cookies.set).toHaveBeenCalledTimes(2);
  });

  it("maps an upstream failure to a gateway error", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      json: async () => ({ error: { message: "Registration unavailable" } }),
    });

    const response = await POST(request({ name: "Acme" }));
    expect(response.status).toBe(502);
    await expect(response.json()).resolves.toMatchObject({
      error: { code: "registration_failed", message: "Registration unavailable" },
    });
  });

  it("does not expose parser or infrastructure errors", async () => {
    const malformed = {
      json: async () => {
        throw new Error("sensitive parser detail");
      },
    } as unknown as Request;

    const response = await POST(malformed);
    expect(response.status).toBe(500);
    await expect(response.json()).resolves.toMatchObject({
      error: {
        code: "unexpected_error",
        message: "Registration service is temporarily unavailable.",
      },
    });
  });
});
