import { POST } from "../route";

// Mock NextResponse
jest.mock("next/server", () => {
  return {
    NextResponse: {
      json: jest.fn((body, init) => {
        return {
          status: init?.status || 200,
          json: async () => body,
        };
      }),
    },
  };
});

// We need to mock fetch
const mockFetch = jest.fn();
global.fetch = mockFetch;

describe("POST /api/tenant/register", () => {
  let mockRequest: Request;

  beforeEach(() => {
    jest.clearAllMocks();
  });

  it("returns 400 if tenant_id or name is missing", async () => {
    mockRequest = {
      json: async () => ({}),
    } as Request;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const res: any = await POST(mockRequest);
    expect(res.status).toBe(400);
    const json = await res.json();
    expect(json.ok).toBe(false);
    expect(json.error.code).toBe("invalid_request");
  });

  it("returns 400 if tenant_id is missing", async () => {
    mockRequest = {
      json: async () => ({ name: "test name" }),
    } as Request;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const res: any = await POST(mockRequest);
    expect(res.status).toBe(400);
  });

  it("returns 400 if name is missing", async () => {
    mockRequest = {
      json: async () => ({ tenant_id: "test-tenant-id" }),
    } as Request;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const res: any = await POST(mockRequest);
    expect(res.status).toBe(400);
  });

  it("returns 502 if upstream registration fails (status not ok, payload ok is undefined)", async () => {
    mockRequest = {
      json: async () => ({ tenant_id: "t1", name: "n1" }),
    } as Request;

    mockFetch.mockResolvedValueOnce({
      ok: false,
      json: async () => ({}),
    });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const res: any = await POST(mockRequest);
    expect(res.status).toBe(502);
    const json = await res.json();
    expect(json.error.code).toBe("registration_failed");
    expect(json.error.message).toBe("Registration failed");
  });

  it("returns 502 if upstream registration fails (status ok, payload ok is false)", async () => {
    mockRequest = {
      json: async () => ({ tenant_id: "t1", name: "n1" }),
    } as Request;

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ ok: false, error: { message: "Internal Error" } }),
    });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const res: any = await POST(mockRequest);
    expect(res.status).toBe(502);
    const json = await res.json();
    expect(json.error.code).toBe("registration_failed");
    expect(json.error.message).toBe("Internal Error");
  });

  it("returns 500 if an unexpected error occurs", async () => {
    mockRequest = {
      json: async () => { throw new Error("JSON parse error") },
    } as Request;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const res: any = await POST(mockRequest);
    expect(res.status).toBe(500);
    const json = await res.json();
    expect(json.error.code).toBe("unexpected_error");
    expect(json.error.message).toBe("JSON parse error");
  });

  it("returns 500 if a non-error object is thrown", async () => {
    mockRequest = {
      json: async () => { throw "A string error" },
    } as Request;

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const res: any = await POST(mockRequest);
    expect(res.status).toBe(500);
    const json = await res.json();
    expect(json.error.code).toBe("unexpected_error");
    expect(json.error.message).toBe("Unexpected error");
  });

  it("returns 200 on success", async () => {
    mockRequest = {
      json: async () => ({ tenant_id: "t1", name: "n1" }),
    } as Request;

    mockFetch.mockResolvedValueOnce({
      ok: true,
      json: async () => ({ ok: true, data: { id: "t1" } }),
    });

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const res: any = await POST(mockRequest);
    expect(res.status).toBe(200);
    const json = await res.json();
    expect(json.ok).toBe(true);
    expect(json.data.id).toBe("t1");
  });
});
