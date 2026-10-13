import { formatInt, formatMoney, formatDate } from "../shared";

describe("shared utility functions", () => {
  describe("formatInt", () => {
    it("formats zero correctly", () => {
      expect(formatInt(0)).toBe("0");
    });

    it("formats positive single-digit and double-digit integers", () => {
      expect(formatInt(7)).toBe("7");
      expect(formatInt(42)).toBe("42");
    });

    it("formats thousands with comma separator", () => {
      expect(formatInt(1000)).toBe("1,000");
      expect(formatInt(1234567)).toBe("1,234,567");
    });

    it("formats negative integers correctly", () => {
      expect(formatInt(-500)).toBe("-500");
      expect(formatInt(-10000)).toBe("-10,000");
    });

    it("formats floating point numbers with comma separation and decimals", () => {
      expect(formatInt(1234.56)).toBe("1,234.56");
    });

    it("formats Number.MAX_SAFE_INTEGER correctly", () => {
      expect(formatInt(Number.MAX_SAFE_INTEGER)).toBe("9,007,199,254,740,991");
    });
  });

  describe("formatMoney", () => {
    it("formats values < 10 with 4 decimal places", () => {
      expect(formatMoney(0)).toBe("0.0000");
      expect(formatMoney(5.1)).toBe("5.1000");
      expect(formatMoney(9.9999)).toBe("9.9999");
    });

    it("formats values >= 10 with 2 decimal places", () => {
      expect(formatMoney(10)).toBe("10.00");
      expect(formatMoney(12.3456)).toBe("12.35");
      expect(formatMoney(100.5)).toBe("100.50");
    });
  });

  describe("formatDate", () => {
    it("formats date strings into US short date format", () => {
      const formatted = formatDate("2025-01-15T12:00:00Z");
      expect(typeof formatted).toBe("string");
      expect(formatted).toContain("2025");
      expect(formatted).toContain("Jan");
    });
  });
});
