import { render, screen, waitFor } from '@testing-library/react';
import ExecutivePage from '../app/executive/page';

global.fetch = async () =>
  Promise.resolve({
    ok: true,
    json: async () => ({
      ok: true,
      status: 'success',
      data: {
        maturity_score: 72,
        grade: 'B',
        roi_multiplier: 2.4,
        total_agents: 12,
        active_agents: 9,
        avg_latency_ms: 210,
        failure_rate_pct: 1.2,
        total_waste_usd: 340.5,
        waste_pct: 8.4,
      },
    }),
  } as unknown as Response);

describe('ExecutivePage', () => {
  it('renders scorecard and refresh button', async () => {
    render(<ExecutivePage />);

    await waitFor(() => screen.getByText('Leadership Scorecard'));

    expect(screen.getByText('Maturity')).toBeTruthy();
    expect(screen.getByText('72/100')).toBeTruthy();
    expect(screen.getByText('Refresh')).toBeTruthy();
  });
});
