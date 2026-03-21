import React, { useEffect, useRef, useImperativeHandle, forwardRef } from 'react'
import * as echarts from 'echarts'
import 'echarts-wordcloud'

export type ColorScheme = 'green' | 'blue' | 'purple' | 'rainbow'
export type ExportFormat = 'png' | 'jpeg'

export interface WordCloudRef {
  exportImage: (format: ExportFormat) => string | null
}

interface WordCloudChartProps {
  data: Array<{ word: string; count: number }>
  colorScheme?: ColorScheme
  sizeRange?: [number, number]
}

const COLOR_SCHEMES: Record<ColorScheme, string[]> = {
  green:   ['#07C160', '#52c41a', '#73d13d', '#95de64', '#389e0d', '#237804', '#b7eb8f'],
  blue:    ['#1677ff', '#40a9ff', '#69c0ff', '#096dd9', '#0050b3', '#4096ff', '#91d5ff'],
  purple:  ['#722ed1', '#9254de', '#b37feb', '#531dab', '#391085', '#c084fc', '#d3adf7'],
  rainbow: ['#f5222d', '#fa8c16', '#fadb14', '#52c41a', '#1677ff', '#722ed1', '#eb2f96'],
}

const WordCloudChart = forwardRef<WordCloudRef, WordCloudChartProps>((
  { data, colorScheme = 'green', sizeRange = [14, 60] },
  ref
) => {
  const chartRef = useRef<HTMLDivElement>(null)
  const chartInstance = useRef<echarts.ECharts | null>(null)

  useImperativeHandle(ref, () => ({
    exportImage: (format: ExportFormat) => {
      if (!chartInstance.current) return null
      return chartInstance.current.getDataURL({
        type: format,
        pixelRatio: 2,
        backgroundColor: '#141414',
      })
    },
  }))

  useEffect(() => {
    if (!chartRef.current || !data || data.length === 0) return

    // 销毁旧实例（配色/尺寸变化时重建，确保布局正确）
    if (chartInstance.current) {
      chartInstance.current.dispose()
      chartInstance.current = null
    }

    const container = chartRef.current
    chartInstance.current = echarts.init(container)

    const colors = COLOR_SCHEMES[colorScheme]

    const wordCloudData = data.map(item => ({ name: item.word, value: item.count }))

    const option = {
      tooltip: {
        show: true,
        formatter: (params: any) => `${params.name}: ${params.value}`,
        backgroundColor: 'rgba(0,0,0,0.85)',
        borderColor: colors[0],
        borderWidth: 1,
        textStyle: { color: '#fff', fontSize: 12 },
      },
      series: [{
        type: 'wordCloud',
        left: 0,
        top: 0,
        width: '100%',
        height: '100%',
        shape: 'circle',
        sizeRange,
        rotationRange: [-45, 45],
        rotationStep: 45,
        gridSize: 8,
        drawOutOfBound: false,
        layoutAnimation: true,
        textStyle: {
          fontFamily: 'sans-serif',
          fontWeight: 'bold',
          color: () => colors[Math.floor(Math.random() * colors.length)],
        },
        emphasis: {
          focus: 'self',
          textStyle: {
            textShadowBlur: 10,
            textShadowColor: colors[0],
          },
        },
        data: wordCloudData,
      }],
    }

    chartInstance.current.setOption(option)

    // 延迟一帧 resize，确保容器尺寸已完全稳定
    const rafId = requestAnimationFrame(() => {
      chartInstance.current?.resize()
    })

    const resizeObserver = new ResizeObserver(() => {
      chartInstance.current?.resize()
    })
    resizeObserver.observe(container)

    return () => {
      cancelAnimationFrame(rafId)
      resizeObserver.disconnect()
    }
  }, [data, colorScheme, sizeRange])

  useEffect(() => {
    return () => {
      chartInstance.current?.dispose()
      chartInstance.current = null
    }
  }, [])

  return (
    <div
      ref={chartRef}
      style={{ width: '100%', height: '100%', minHeight: 200 }}
    />
  )
})

WordCloudChart.displayName = 'WordCloudChart'
export default WordCloudChart
