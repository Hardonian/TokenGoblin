import { POST } from "../route";

// Mock the global fetch
global.fetch = jest.fn();

describe("POST /api/tenant/register", () => {
  const originalEnv = process.env;

  beforeEach(() => {
    jest.clearAllMocks();
    process.env = { ...originalEnv };
  });

  afterAll(() => {
    process.env = originalEnv;
  });

  function createMockRequest(body: any): Request {
    return {
      json: jest.fn().mockResolvedValue(body),
    } as unknown as Request;
  }

  it("returns 400 if tenant_id is missing", async () => {
    const req = createMockRequest({ name: "Test Tenant" });
    const response = await POST(req);
    expect(response.status).toBe(400);
    const data = await response.json();
    expect(data.ok).toBe(false);
    expect(data.error.code).toBe("invalid_request");
  });

  it("returns 400 if name is missing", async () => {
    const req = createMockRequest({ tenant_id: "t1" });
    const response = await POST(req);
    expect(response.status).toBe(400);
  });

  it("returns 200 on successful registration", async () => {
    const mockResponse = { ok: true, data: { tenant_id: "t1" } };
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: true,
      json: jest.fn().mockResolvedValue(mockResponse),
    });

    const req = createMockRequest({ tenant_id: "t1", name: "Test Tenant" });
    const response = await POST(req);

    expect(response.status).toBe(200);
    const data = await response.json();
    expect(data.ok).toBe(true);
    expect(data.data.tenant_id).toBe("t1");
    expect(global.fetch).toHaveBeenCalledWith(
      "http://localhost:8080/api/tenant/register",
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ tenant_id: "t1", name: "Test Tenant" }),
      })
    );
  });

  it("returns 502 if upstream fails", async () => {
    const mockResponse = { ok: false, error: { message: "Upstream error" } };
    (global.fetch as jest.Mock).mockResolvedValue({
      ok: false,
      json: jest.fn().mockResolvedValue(mockResponse),
    });

    const req = createMockRequest({ tenant_id: "t1", name: "Test Tenant" });
    const response = await POST(req);

    expect(response.status).toBe(502);
    const data = await response.json();
    expect(data.ok).toBe(false);
    expect(data.error.code).toBe("registration_failed");
    expect(data.error.message).toBe("Upstream error");
  });

  it("returns 500 on unexpected error", async () => {
    const req = {
      json: jest.fn().mockRejectedValue(new Error("JSON parsing error")),
    } as unknown as Request;

    const response = await POST(req);

    expect(response.status).toBe(500);
    const data = await response.json();
    expect(data.ok).toBe(false);
    expect(data.error.code).toBe("unexpected_error");
    expect(data.error.message).toBe("JSON parsing error");
  });
});
