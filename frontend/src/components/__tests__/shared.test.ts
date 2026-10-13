import { formatMoney, formatInt, formatDate, formatDateTime } from "../shared";

describe("shared formatting utilities", () => {
  describe("formatMoney", () => {
    it("should format values >= 10 with 2 decimal places", () => {
      expect(formatMoney(10)).toBe("10.00");
      expect(formatMoney(10.5)).toBe("10.50");
      expect(formatMoney(12.3456)).toBe("12.35");
      expect(formatMoney(1234.5678)).toBe("1234.57");
    });

    it("should format values < 10 with 4 decimal places", () => {
      expect(formatMoney(0)).toBe("0.0000");
      expect(formatMoney(0.12345)).toBe("0.1235");
      expect(formatMoney(9.9999)).toBe("9.9999");
      expect(formatMoney(1.2)).toBe("1.2000");
    });

    it("should handle negative numbers correctly according to threshold", () => {
      expect(formatMoney(-5)).toBe("-5.0000");
      expect(formatMoney(-10)).toBe("-10.0000");
    });
  });

  describe("formatInt", () => {
    it("should format integers with comma separators", () => {
      expect(formatInt(1000)).toBe("1,000");
      expect(formatInt(1000000)).toBe("1,000,000");
      expect(formatInt(42)).toBe("42");
    });
  });

  describe("formatDate", () => {
    it("should format date strings into short month day year format", () => {
      const formatted = formatDate("2025-01-15T12:00:00Z");
      expect(formatted).toContain("2025");
      expect(formatted).toContain("Jan");
    });
  });

  describe("formatDateTime", () => {
    it("should format date-time strings into month day year time format", () => {
      const formatted = formatDateTime("2025-01-15T12:34:00Z");
      expect(formatted).toContain("2025");
      expect(formatted).toContain("Jan");
    });
  });
});
