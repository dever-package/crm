import { useEffect, useRef } from "react";
import type { MutableRefObject, ReactNode } from "react";
import * as echarts from "echarts";
import type { EChartsOption } from "echarts";

export type { EChartsOption } from "echarts";

export const crmChartTextColor = "#64748b";
export const crmChartAxisColor = "#e2e8f0";
export const crmChartSplitLineColor = "#e5e7eb";

type CrmChartInstance = ReturnType<typeof echarts.init>;

export function CrmEChart({
  option,
  height = 300,
  minWidth = 560,
  empty,
  isEmpty = false,
  ariaLabel,
}: {
  option: EChartsOption;
  height?: number;
  minWidth?: number;
  empty?: ReactNode;
  isEmpty?: boolean;
  ariaLabel?: string;
}) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const chartRef = useRef<CrmChartInstance | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || isEmpty) {
      disposeCrmChart(chartRef);
      return;
    }

    const chart =
      chartRef.current ||
      echarts.init(container, undefined, { renderer: "canvas" });
    chartRef.current = chart;
    chart.setOption(option, true);

    chart.resize();
    const frame = window.requestAnimationFrame(() => chart.resize());
    return () => window.cancelAnimationFrame(frame);
  }, [isEmpty, option]);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || isEmpty) return;

    const resize = () => chartRef.current?.resize();
    const observer =
      typeof ResizeObserver !== "undefined" ? new ResizeObserver(resize) : null;
    observer?.observe(container);
    window.addEventListener("resize", resize);

    return () => {
      observer?.disconnect();
      window.removeEventListener("resize", resize);
    };
  }, [isEmpty]);

  useEffect(() => () => disposeCrmChart(chartRef), []);

  if (isEmpty) {
    return <>{empty}</>;
  }

  return (
    <div className="overflow-x-auto">
      <div
        ref={containerRef}
        role="img"
        aria-label={ariaLabel}
        style={{ height, minWidth, width: "100%" }}
      />
    </div>
  );
}

function disposeCrmChart(chartRef: MutableRefObject<CrmChartInstance | null>) {
  if (!chartRef.current) return;
  chartRef.current.dispose();
  chartRef.current = null;
}
