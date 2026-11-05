import { useMemo, useState, type ReactNode } from 'react'
import { ArrowDown, ArrowUp } from 'lucide-react'
import { cx } from '@/lib/format'
import { Empty } from './primitives'

export interface Column<T> {
  key: string
  header: string
  render: (row: T) => ReactNode
  /** Provide a sort key to make the column sortable. */
  sort?: (row: T) => string | number
  align?: 'right'
  className?: string
}

interface Props<T> {
  columns: Column<T>[]
  rows: T[]
  rowKey: (row: T) => string
  onRowClick?: (row: T) => void
  selectedKey?: string
  emptyTitle?: string
  caption: string
}

export function DataTable<T>({ columns, rows, rowKey, onRowClick, selectedKey, emptyTitle = 'Nothing to show', caption }: Props<T>) {
  const [sort, setSort] = useState<{ key: string; dir: 1 | -1 } | null>(null)

  const sorted = useMemo(() => {
    if (!sort) return rows
    const col = columns.find((c) => c.key === sort.key)
    if (!col?.sort) return rows
    const get = col.sort
    return [...rows].sort((a, b) => {
      const x = get(a)
      const y = get(b)
      return (x < y ? -1 : x > y ? 1 : 0) * sort.dir
    })
  }, [rows, sort, columns])

  const toggle = (key: string) => setSort((s) => (s?.key === key ? (s.dir === 1 ? { key, dir: -1 } : null) : { key, dir: 1 }))

  if (rows.length === 0) return <Empty title={emptyTitle} />

  return (
    <div className="table-wrap">
      <table className="tbl">
        <caption className="sr-only">{caption}</caption>
        <thead>
          <tr>
            {columns.map((c) => {
              const active = sort?.key === c.key
              return (
                <th key={c.key} scope="col" className={cx(c.sort && 'sortable', c.align === 'right' && 'right')} aria-sort={active ? (sort.dir === 1 ? 'ascending' : 'descending') : undefined}>
                  {c.sort ? (
                    <button type="button" className="th-btn" onClick={() => toggle(c.key)}>
                      {c.header}
                      {active && (sort.dir === 1 ? <ArrowUp size={12} /> : <ArrowDown size={12} />)}
                    </button>
                  ) : (
                    c.header
                  )}
                </th>
              )
            })}
          </tr>
        </thead>
        <tbody>
          {sorted.map((row) => {
            const k = rowKey(row)
            return (
              <tr
                key={k}
                className={cx(onRowClick && 'clickable', selectedKey === k && 'selected')}
                tabIndex={onRowClick ? 0 : undefined}
                onClick={onRowClick ? () => onRowClick(row) : undefined}
                onKeyDown={onRowClick ? (e) => e.key === 'Enter' && onRowClick(row) : undefined}
              >
                {columns.map((c) => (
                  <td key={c.key} className={cx(c.align === 'right' && 'right', c.className)}>
                    {c.render(row)}
                  </td>
                ))}
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
