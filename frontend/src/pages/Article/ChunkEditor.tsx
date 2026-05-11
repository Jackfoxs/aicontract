import { useEffect, useMemo, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Table, Button, Space, Modal, Input, Message, Popconfirm } from '@arco-design/web-react'
import { IconLeft, IconPlus, IconDelete, IconEdit, IconToTop, IconToBottom, IconScissor, IconSort } from '@arco-design/web-react/icon'
import { ChunkItem, getChunks, updateChunk, splitChunk, mergeChunks, reorderChunks, deleteChunk } from '@/api/document'

const TextArea = Input.TextArea

export default function ChunkEditor() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [rows, setRows] = useState<ChunkItem[]>([])
  const [selected, setSelected] = useState<string[]>([])

  const [editVisible, setEditVisible] = useState(false)
  const [editing, setEditing] = useState<ChunkItem | null>(null)
  const [editTitle, setEditTitle] = useState('')
  const [editContent, setEditContent] = useState('')

  const load = async () => {
    if (!id) return
    setLoading(true)
    try {
      const res = await getChunks(id)
      const list = (res?.data?.list || []).slice().sort((a, b) => (a.order_index - b.order_index) || (Number(a.id) - Number(b.id)))
      // 若后端未设置排序或存在重复，前端按当前顺序临时填充，保证展示与移动可用
      const set = new Set(list.map(i => i.order_index))
      const needFill = set.size < list.length || list.every(i => i.order_index === 0)
      if (needFill) {
        list.forEach((it, idx) => { (it as any).order_index = idx })
      }
      setRows(list)
    } catch (e:any) {
      Message.error(e?.message || '加载切片失败')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [id])

  const openEdit = (row: ChunkItem) => {
    setEditing(row)
    setEditTitle(row.title || '')
    setEditContent(row.content || '')
    setEditVisible(true)
  }

  const saveEdit = async () => {
    if (!editing) return
    try {
      await updateChunk(editing.chunk_id, { title: editTitle, content: editContent })
      Message.success('已保存并重建向量')
      setEditVisible(false)
      setEditing(null)
      load()
    } catch (e:any) {
      Message.error(e?.message || '保存失败')
    }
  }

  const doSplit = async (row: ChunkItem) => {
    try {
      await splitChunk(row.chunk_id)
      Message.success('按“第X条”拆分完成')
      load()
    } catch (e:any) {
      Message.error(e?.message || '拆分失败')
    }
  }

  const doMergeSelected = async () => {
    if (selected.length < 2) {
      Message.warning('请至少选择两条合并')
      return
    }
    try {
      await mergeChunks(selected)
      Message.success('合并完成')
      setSelected([])
      load()
    } catch (e:any) {
      Message.error(e?.message || '合并失败')
    }
  }

  // 解析标题中的“第X条”序号，优先阿拉伯数字，其次中文数字；无法解析返回 Infinity
  const parseArticleIndex = (title: string): number => {
    if (!title) return Number.POSITIVE_INFINITY
    const m = title.match(/第([0-9一二三四五六七八九十百零〇]+)条/)
    if (!m) return Number.POSITIVE_INFINITY
    const token = (m[1] || '').trim()
    const n = parseInt(token, 10)
    if (!Number.isNaN(n)) return n
    const digitMap: Record<string, number> = { '零':0,'〇':0,'一':1,'二':2,'三':3,'四':4,'五':5,'六':6,'七':7,'八':8,'九':9 }
    // 简易中文数字转阿拉伯（支持至百位）
    let rest = token
    let val = 0
    const idxB = rest.indexOf('百')
    if (idxB >= 0) {
      const h = idxB === 0 ? 1 : (digitMap[rest[idxB-1]] || 0)
      val += h * 100
      rest = rest.slice(idxB + 1)
    }
    const idxT = rest.indexOf('十')
    if (idxT >= 0) {
      const t = idxT === 0 ? 1 : (digitMap[rest[idxT-1]] || 0)
      val += t * 10
      rest = rest.slice(idxT + 1)
    }
    // 单位
    if (rest.length > 0) {
      const u = digitMap[rest[rest.length - 1]] || 0
      val += u
    }
    return val > 0 ? val : Number.POSITIVE_INFINITY
  }

  const sortByArticleOrder = async () => {
    const sorted = rows.slice().sort((a, b) => parseArticleIndex(a.title) - parseArticleIndex(b.title))
    setRows(sorted)
    // 同步后端排序
    try {
      const orders = sorted.map((r, idx) => ({ chunk_id: r.chunk_id, order_index: idx }))
      await reorderChunks(orders)
      Message.success('已按“第X条”排序')
    } catch (e:any) {
      Message.error(e?.message || '排序保存失败')
    }
  }

  const move = async (row: ChunkItem, dir: 'up'|'down') => {
    const idx = rows.findIndex(r => r.chunk_id === row.chunk_id)
    if (idx < 0) return
    const targetIdx = dir === 'up' ? idx - 1 : idx + 1
    if (targetIdx < 0 || targetIdx >= rows.length) return
    const list = rows.slice()
    const a = list[idx]
    const b = list[targetIdx]
    const orders = [
      { chunk_id: a.chunk_id, order_index: b.order_index },
      { chunk_id: b.chunk_id, order_index: a.order_index }
    ]
    try {
      await reorderChunks(orders)
      load()
    } catch (e:any) {
      Message.error(e?.message || '调整顺序失败')
    }
  }

  const doDelete = async (row: ChunkItem) => {
    try {
      await deleteChunk(row.chunk_id)
      Message.success('已删除')
      load()
    } catch (e:any) {
      Message.error(e?.message || '删除失败')
    }
  }

  const columns = useMemo(() => [
    { title: '序', dataIndex: 'serial', width: 60, render: (_:any, __:ChunkItem, index:number) => index + 1 },
    { 
      title: (
        <Space size={4}>
          <span>标题</span>
          <Button size="mini" shape="circle" type="text" icon={<IconSort />} onClick={sortByArticleOrder} />
        </Space>
      ),
      dataIndex: 'title',
      width: 220,
      render: (_:any, r:ChunkItem) => <div style={{whiteSpace:'nowrap',overflow:'hidden',textOverflow:'ellipsis'}}>{r.title || '-'}</div>
    },
    { title: '内容预览', dataIndex: 'content', render: (_:any, r:ChunkItem) => <div style={{maxHeight:96,overflow:'hidden',whiteSpace:'pre-wrap'}}>{r.content}</div> },
    { title: '操作', dataIndex: 'op', width: 240, render: (_:any, r:ChunkItem) => (
      <Space>
        <Button size="mini" icon={<IconEdit />} onClick={() => openEdit(r)}>编辑</Button>
        <Button size="mini" icon={<IconScissor />} onClick={() => doSplit(r)}>按“第X条”拆分</Button>
        <Button size="mini" icon={<IconToTop />} onClick={() => move(r,'up')}>上移</Button>
        <Button size="mini" icon={<IconToBottom />} onClick={() => move(r,'down')}>下移</Button>
        <Popconfirm focusLock title="确认删除该切片？" onOk={() => doDelete(r)}>
          <Button size="mini" status="danger" icon={<IconDelete />}>删除</Button>
        </Popconfirm>
      </Space>
    )}
  ], [rows])

  return (
    <div style={{ padding: 16 }}>
      <Space style={{ marginBottom: 12 }}>
        <Button icon={<IconLeft />} onClick={() => navigate(`/articles/${id}`)}>返回</Button>
        <Button type="primary" icon={<IconPlus />} onClick={doMergeSelected} disabled={selected.length < 2}>合并所选</Button>
      </Space>

      <Table
        rowKey="chunk_id"
        loading={loading}
        columns={columns as any}
        data={rows}
        pagination={false}
        rowSelection={{
          selectedRowKeys: selected,
          onChange: (keys) => setSelected(keys as string[])
        }}
        border={false}
      />

      <Modal
        visible={editVisible}
        title="编辑切片"
        onCancel={() => setEditVisible(false)}
        onOk={saveEdit}
        style={{ width: 900 }}
      >
        <Space direction="vertical" style={{ width: '100%' }}>
          <Input placeholder="标题" value={editTitle} onChange={setEditTitle} />
          <TextArea rows={14} value={editContent} onChange={setEditContent} />
        </Space>
      </Modal>
    </div>
  )
}


