import { buildCheckoutSuccessURL, buildCheckoutCancelURL } from '../billing';

describe('billing URLs', () => {
  describe('buildCheckoutSuccessURL', () => {
    it('builds success URL correctly', () => {
      expect(buildCheckoutSuccessURL('http://localhost:3000', 'pro')).toBe('http://localhost:3000/billing?plan=pro');
    });

    it('removes trailing slash from origin', () => {
      expect(buildCheckoutSuccessURL('http://localhost:3000/', 'pro')).toBe('http://localhost:3000/billing?plan=pro');
    });

    it('encodes plan correctly', () => {
      expect(buildCheckoutSuccessURL('http://localhost:3000', 'pro plan')).toBe('http://localhost:3000/billing?plan=pro%20plan');
    });
  });

  describe('buildCheckoutCancelURL', () => {
    it('builds cancel URL correctly', () => {
      expect(buildCheckoutCancelURL('http://localhost:3000', 'pro')).toBe('http://localhost:3000/pricing?plan=pro');
    });

    it('removes trailing slash from origin', () => {
      expect(buildCheckoutCancelURL('http://localhost:3000/', 'pro')).toBe('http://localhost:3000/pricing?plan=pro');
    });

    it('encodes plan correctly', () => {
      expect(buildCheckoutCancelURL('http://localhost:3000', 'pro plan')).toBe('http://localhost:3000/pricing?plan=pro%20plan');
    });
  });
});
