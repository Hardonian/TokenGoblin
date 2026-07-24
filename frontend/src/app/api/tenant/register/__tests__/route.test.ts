/**
 * @jest-environment node
 */
import { POST } from "../route";

const mockFetch = jest.fn();
global.fetch = mockFetch;

describe("POST /api/tenant/register", () => {
  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("returns 400 if tenant_id is missing", async () => {
    const req = new Request("http://localhost", {
      method: "POST",
      body: JSON.stringify({ name: "Test Tenant" }),
    });

    const res = await POST(req);
    expect(res.status).toBe(400);
    const json = await res.json();
    expect(json.ok).toBe(false);
    expect(json.error.code).toBe("invalid_request");
  });

  it("returns 400 if name is missing", async () => {
    const req = new Request("http://localhost", {
      method: "POST",
      body: JSON.stringify({ tenant_id: "tenant-123" }),
    });

    const res = await POST(req);
    expect(res.status).toBe(400);
    const json = await res.json();
    expect(json.ok).toBe(false);
    expect(json.error.code).toBe("invalid_request");
  });

  it("returns 200 on successful registration", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: jest.fn().mockResolvedValue({
        ok: true,
        data: { id: "tenant-123", name: "Test Tenant" },
      }),
    });

    const req = new Request("http://localhost", {
      method: "POST",
      body: JSON.stringify({ tenant_id: "tenant-123", name: "Test Tenant" }),
    });

    const res = await POST(req);
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.ok).toBe(true);
    expect(json.data.id).toBe("tenant-123");

    expect(mockFetch).toHaveBeenCalledTimes(1);
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/api/tenant/register"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ tenant_id: "tenant-123", name: "Test Tenant" }),
      })
    );
  });

  it("returns 502 when upstream registration fails", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      json: jest.fn().mockResolvedValue({
        ok: false,
        error: { message: "Upstream error" },
      }),
    });

    const req = new Request("http://localhost", {
      method: "POST",
      body: JSON.stringify({ tenant_id: "tenant-123", name: "Test Tenant" }),
    });

    const res = await POST(req);
    expect(res.status).toBe(502);
    const json = await res.json();
    expect(json.ok).toBe(false);
    expect(json.error.code).toBe("registration_failed");
    expect(json.error.message).toBe("Upstream error");
  });

  it("returns 500 when fetch throws an error", async () => {
    mockFetch.mockRejectedValueOnce(new Error("Network failure"));

    const req = new Request("http://localhost", {
      method: "POST",
      body: JSON.stringify({ tenant_id: "tenant-123", name: "Test Tenant" }),
    });

    const res = await POST(req);
    expect(res.status).toBe(500);
    const json = await res.json();
    expect(json.ok).toBe(false);
    expect(json.error.code).toBe("unexpected_error");
    expect(json.error.message).toBe("Network failure");
  });
});
