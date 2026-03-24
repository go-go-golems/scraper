import type { Meta, StoryObj } from '@storybook/react-vite';
import { ThroughputChart } from './ThroughputChart';
import type { ThroughputSeries } from './ThroughputChart';

const meta: Meta<typeof ThroughputChart> = {
  title: 'Queues/ThroughputChart',
  component: ThroughputChart,
};

export default meta;
type Story = StoryObj<typeof ThroughputChart>;

function makeTimeLabels(count: number): string[] {
  return Array.from({ length: count }, (_, i) => {
    const min = count - 1 - i;
    return `${String(Math.floor(min / 60)).padStart(2, '0')}:${String(min % 60).padStart(2, '0')}`;
  }).reverse();
}

function makeSeries(queueKey: string, values: number[]): ThroughputSeries {
  const times = makeTimeLabels(values.length);
  return {
    queueKey,
    points: values.map((v, i) => ({ time: times[i], opsPerMin: v })),
  };
}

const defaultData: ThroughputSeries[] = [
  makeSeries('site:hn:http', [12, 15, 14, 18, 20, 19, 22, 18, 16, 14, 17, 21, 23, 20, 18]),
  makeSeries('site:hn:js', [8, 10, 9, 12, 11, 13, 14, 12, 10, 9, 11, 13, 15, 12, 10]),
  makeSeries('site:sd:http', [4, 5, 6, 5, 7, 6, 8, 7, 5, 4, 6, 7, 8, 6, 5]),
];

export const Default: Story = {
  args: {
    data: defaultData,
    timeRange: '15m',
  },
};

export const SingleQueue: Story = {
  args: {
    data: [
      makeSeries('site:hn:http', [10, 12, 11, 14, 13, 15, 16, 14, 12, 11, 13, 15, 17, 14, 12]),
    ],
    timeRange: '15m',
  },
};

export const Bursty: Story = {
  args: {
    data: [
      makeSeries('site:hn:http', [2, 45, 3, 1, 50, 4, 2, 48, 5, 1, 42, 3, 2, 47, 4]),
      makeSeries('site:sd:http', [1, 30, 2, 0, 35, 1, 0, 32, 2, 1, 28, 0, 1, 33, 2]),
    ],
    timeRange: '15m',
  },
};

export const Idle: Story = {
  args: {
    data: [
      makeSeries('site:hn:http', [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
      makeSeries('site:hn:js', [0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0]),
    ],
    timeRange: '5m',
  },
};
