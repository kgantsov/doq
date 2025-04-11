import { Chart, useChart } from "@chakra-ui/charts";
import { Area, AreaChart } from "recharts";

const SparkLine = ({ data }: { data: { value: number }[] }) => {
  const chart = useChart({
    data: data,
    series: [{ color: "teal.solid" }],
  });

  if (!data || data.length < 2) {
    return null;
  }

  return (
    <Chart.Root height="10" chart={chart}>
      <AreaChart
        data={chart.data}
        margin={{ top: 0, right: 0, left: 0, bottom: 0 }}
      >
        {chart.series.map((item, i) => {
          return (
            <Area
              key={`${item.name}-${i}`}
              isAnimationActive={false}
              dataKey={chart.key(item.name)}
              fill={chart.color(item.color)}
              fillOpacity={0.2}
              stroke={chart.color(item.color)}
              strokeWidth={2}
            />
          );
        })}
      </AreaChart>
    </Chart.Root>
  );
};

export default SparkLine;
