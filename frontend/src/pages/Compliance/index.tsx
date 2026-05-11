import { useEffect, useMemo, useState } from 'react'
import {
  Button,
  Card,
  Grid,
  Message,
  Space,
  Tabs,
  Tag,
  Typography,
  Upload,
  Progress,
  Pagination,
  Modal,
  Table,
  Input,
} from '@arco-design/web-react'
import type { TableColumnProps } from '@arco-design/web-react'
import { IconCloudDownload, IconRefresh } from '@arco-design/web-react/icon'
import { useComplianceStore } from '@/store'
import type {
  ComplianceJob,
  ComplianceJobDetail,
  ComplianceHighlight,
  ComplianceIssue,
} from '@/store/complianceStore'
import {
  submitComplianceJob,
  fetchComplianceJobs,
  fetchComplianceJobDetail,
  fetchComplianceIssues,
  fetchComplianceHighlights,
  retryComplianceJob,
  downloadComplianceReport,
  uploadComplianceFile,
  fetchComplianceRules,
} from '@/api'
import type { ComplianceRuleOption, ComplianceRuleQuery, UploadComplianceFileResponse } from '@/api'
import './index.css'

const { Row, Col } = Grid
const TabPane = Tabs.TabPane
type UploadedFileInfo = UploadComplianceFileResponse

const STATUS_META: Record<string, { label: string; color: string }> = {
  pending: { label: '等待中', color: 'arcoblue' },
  running: { label: '执行中', color: 'orange' },
  success: { label: '成功', color: 'green' },
  failed: { label: '失败', color: 'red' },
  matched: { label: '满足', color: 'green' },
  partial: { label: '部分满足', color: 'gold' },
  inconsistent: { label: '不一致', color: 'magenta' },
  missing: { label: '缺失', color: 'red' },
}

function getStatusMeta(status: string) {
  return STATUS_META[status] ?? { label: status || '未知', color: 'gray' }
}

export default function CompliancePage() {
  const {
    jobs,
    pagination,
    currentJobId,
    currentDetail,
    highlights,
    loading,
    issueLoading,
    highlightLoading,
    setJobs,
    setCurrentJobId,
    setDetail,
    setHighlights,
    setLoading,
    setIssueLoading,
    setHighlightLoading,
  } = useComplianceStore()

  const [submitting, setSubmitting] = useState(false)
  const [uploadingRole, setUploadingRole] = useState<'tender' | 'response' | null>(null)
  const [tenderFile, setTenderFile] = useState<UploadedFileInfo | null>(null)
  const [responseFile, setResponseFile] = useState<UploadedFileInfo | null>(null)
  const [activeHighlightRole, setActiveHighlightRole] = useState<'response' | 'tender'>('response')
  const [selectedRuleIds, setSelectedRuleIds] = useState<string[]>([])
  const [ruleDictionary, setRuleDictionary] = useState<Record<string, ComplianceRuleOption>>({})
  const [ruleModalVisible, setRuleModalVisible] = useState(false)
  const [ruleModalLoading, setRuleModalLoading] = useState(false)
  const [ruleModalData, setRuleModalData] = useState<ComplianceRuleOption[]>([])
  const [ruleModalSelectedKeys, setRuleModalSelectedKeys] = useState<string[]>([])
  const [ruleModalKeyword, setRuleModalKeyword] = useState('')
  const [rulePreview, setRulePreview] = useState<ComplianceRuleOption | null>(null)

  useEffect(() => {
    loadJobs(pagination.page, pagination.pageSize)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  const loadJobs = async (page: number, pageSize: number) => {
    try {
      setLoading(true)
      const res = await fetchComplianceJobs({ page, page_size: pageSize })
      const data = res.data
      const jobsForStore = (data?.list ?? []).map((item) => ({
        ...item,
        status: item.status as ComplianceJob['status'],
      })) as ComplianceJob[]
      setJobs(jobsForStore, data?.total ?? 0, page, pageSize)
    } catch (error: any) {
      Message.error(error?.message || '获取任务列表失败')
    } finally {
      setLoading(false)
    }
  }

  const loadRuleOptions = async (keyword: string, withContent = false) => {
    if (withContent) {
      setRuleModalLoading(true)
    }
    try {
      const params: ComplianceRuleQuery & { include_content?: number } = {
        page: 1,
        page_size: withContent ? 100 : 50,
      }
      if (keyword) {
        params.keyword = keyword
      }
      if (withContent) {
        params.include_content = 1
      }
      const res = await fetchComplianceRules(params)
      const list = res.data?.list ?? []
      if (withContent) {
        setRuleModalData(list)
      }
      if (list.length) {
        setRuleDictionary((prev) => {
          const next = { ...prev }
          list.forEach((item) => {
            next[item.id] = item
          })
          return next
        })
      }
    } catch (error: any) {
      Message.error(error?.message || '加载规范条目失败')
    } finally {
      if (withContent) {
        setRuleModalLoading(false)
      }
    }
  }

  const handleOpenRuleModal = () => {
    setRuleModalVisible(true)
    setRuleModalSelectedKeys(selectedRuleIds)
    setRuleModalKeyword('')
    loadRuleOptions('', true)
  }

  const handleRuleSearch = (value: string) => {
    const keyword = value.trim()
    setRuleModalKeyword(keyword)
    loadRuleOptions(keyword, true)
  }

  const handleRuleModalConfirm = () => {
    const uniqueKeys = Array.from(new Set(ruleModalSelectedKeys))
    setSelectedRuleIds(uniqueKeys)
    setRuleModalVisible(false)
  }

  const handleSelectAllRules = () => {
    setRuleModalSelectedKeys(ruleModalData.map((item) => item.id))
  }

  const handleClearModalSelection = () => {
    setRuleModalSelectedKeys([])
  }

  const handleRuleRemove = (id: string) => {
    setSelectedRuleIds((prev) => prev.filter((item) => item !== id))
  }

  const handleClearSelectedRules = () => {
    setSelectedRuleIds([])
  }

  const ruleModalColumns = useMemo<TableColumnProps<ComplianceRuleOption>[]>(
    () => [
      {
        title: '规则ID',
        dataIndex: 'rule_id',
        width: 180,
        ellipsis: true,
        render: (value: string | undefined) => value || '-'
      },
      {
        title: '标题',
        dataIndex: 'title',
        width: 200,
        ellipsis: true,
        render: (value: string | undefined, record: ComplianceRuleOption) =>
          value || record.rule_id || '未命名条目'
      },
      {
        title: '章节',
        dataIndex: 'section_path',
        width: 160,
        ellipsis: true,
        render: (value: string | undefined) => value || '-'
      },
      {
        title: '内容概览',
        dataIndex: 'content',
        render: (_: unknown, record: ComplianceRuleOption) => (
          <Typography.Paragraph style={{ marginBottom: 0 }} ellipsis={{ rows: 2 }}>
            {record.content || record.content_preview || '-'}
          </Typography.Paragraph>
        )
      },
      {
        title: '操作',
        dataIndex: 'id',
        width: 100,
        render: (_: unknown, record: ComplianceRuleOption) => (
          <Button type="text" size="small" onClick={() => setRulePreview(record)}>
            查看原文
          </Button>
        )
      },
    ],
    []
  )

  const loadDetail = async (jobId: string) => {
    try {
      setIssueLoading(true)
      const [detailRes, issuesRes] = await Promise.all([
        fetchComplianceJobDetail(jobId),
        fetchComplianceIssues(jobId),
      ])
      const detailData = detailRes.data
      const issueData = (
        (issuesRes.data as any)?.list ??
        issuesRes.data ??
        detailData?.issues ??
        []
      ) as ComplianceIssue[]

      if (detailData?.job) {
        setDetail({ job: detailData.job as ComplianceJobDetail['job'], issues: issueData })
      } else {
        setDetail(undefined)
      }
    } catch (error: any) {
      Message.error(error?.message || '获取任务详情失败')
    } finally {
      setIssueLoading(false)
    }
  }

  const loadHighlights = async (jobId: string, fileRole: 'tender' | 'response') => {
    try {
      setHighlightLoading(true)
      const res = await fetchComplianceHighlights(jobId, { file_role: fileRole })
      setHighlights(res.data || [])
    } catch (error: any) {
      Message.error(error?.message || '获取高亮失败')
    } finally {
      setHighlightLoading(false)
    }
  }

  const handleSelectJob = (jobId: string) => {
    setCurrentJobId(jobId)
    setActiveHighlightRole('response')
    loadDetail(jobId)
    loadHighlights(jobId, 'response')
  }

  const handleRetry = async (jobId: string) => {
    try {
      await retryComplianceJob(jobId)
      Message.success('任务已重新排队')
      loadJobs(pagination.page, pagination.pageSize)
    } catch (error: any) {
      Message.error(error?.message || '重试失败')
    }
  }

  const handleDownload = async (jobId: string, format: 'json' | 'csv' | 'pdf') => {
    try {
      const res = await downloadComplianceReport(jobId, format)
      const contentType = res.headers?.['content-type'] || 'application/octet-stream'
      const blob = new Blob([res.data], { type: contentType })
      const url = window.URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `compliance-report-${jobId}.${format}`
      a.click()
      window.URL.revokeObjectURL(url)
    } catch (error: any) {
      Message.error(error?.message || '下载失败')
    }
  }

  const handleFileUpload = (role: 'tender' | 'response') => async (options: any) => {
    const originFile: File = (options.file?.originFile as File) || (options.file as File)
    if (!originFile) {
      Message.error('未获取到上传文件')
      options.onError?.(new Error('missing file'))
      return
    }
    const formData = new FormData()
    formData.append('file', originFile)
    formData.append('role', role)

    try {
      setUploadingRole(role)
      const res = await uploadComplianceFile(formData)
      const data = res.data
      if (!data) {
        throw new Error('服务器未返回解析结果')
      }
      if (role === 'tender') {
        setTenderFile(data)
      } else {
        setResponseFile(data)
      }
      Message.success(`${role === 'tender' ? '比选' : '响应'}文件上传并解析成功`)
      options.onSuccess?.(data, options.file)
    } catch (error: any) {
      Message.error(error?.message || '上传失败')
      options.onError?.(error as Error)
    } finally {
      setUploadingRole(null)
    }
  }

  const getUploadProps = (role: 'tender' | 'response') => ({
    showUploadList: false,
    accept: '.pdf,.doc,.docx',
    customRequest: handleFileUpload(role),
  })

  const canSubmit = useMemo(
    () => Boolean(tenderFile?.file_id && responseFile?.file_id),
    [tenderFile, responseFile],
  )

  const handleSubmit = async () => {
    if (!tenderFile?.file_id || !responseFile?.file_id) {
      Message.warning('请上传比选文件和响应文件')
      return
    }
    try {
      setSubmitting(true)
      await submitComplianceJob({
        tender_file_id: tenderFile.file_id,
        response_file_id: responseFile.file_id,
        selected_rule_ids: selectedRuleIds,
      })
      Message.success('任务提交成功')
      loadJobs(pagination.page, pagination.pageSize)
      setSelectedRuleIds([])
    } catch (error: any) {
      Message.error(error?.message || '提交失败')
    } finally {
      setSubmitting(false)
    }
  }

  const renderUploadedInfo = (file: UploadedFileInfo | null, role: 'tender' | 'response') => {
    if (!file) return null
    const color = role === 'tender' ? 'arcoblue' : 'green'
    const quality = Number.isFinite(file.quality_score)
      ? `${(file.quality_score * 100).toFixed(1)}%`
      : '未知'

    return (
      <div className="upload-preview">
        <Space direction="vertical" size={4} style={{ width: '100%' }}>
          <Space wrap>
            <Tag color={color}>日志ID: {file.file_id}</Tag>
            <Typography.Text type="secondary">{file.file_name}</Typography.Text>
          </Space>
          <Typography.Text type="secondary">
            质量评分：{quality} · 文本长度：{file.text_length}
            {file.page_count ? ` · 页数：${file.page_count}` : ''}
          </Typography.Text>
          {file.content_preview && (
            <Typography.Paragraph type="secondary" style={{ marginBottom: 0 }} ellipsis={{ rows: 2 }}>
              {file.content_preview}
            </Typography.Paragraph>
          )}
          <Button
            size="mini"
            type="text"
            onClick={() => (role === 'tender' ? setTenderFile(null) : setResponseFile(null))}
          >
            重新选择
          </Button>
        </Space>
      </div>
    )
  }

  return (
    <div className="compliance-page">
      <Row gutter={16} className="compliance-main">
        <Col span={7} className="compliance-left">
          <Card title="提交任务" className="compliance-card">
            <Space direction="vertical" size="large" style={{ width: '100%' }}>
              <div>
                <Typography.Text>比选文件</Typography.Text>
                <Upload {...getUploadProps('tender')}>
                  <Button type="outline" loading={uploadingRole === 'tender'}>上传比选文件</Button>
                </Upload>
                {renderUploadedInfo(tenderFile, 'tender')}
              </div>
              <div>
                <Typography.Text>响应文件</Typography.Text>
                <Upload {...getUploadProps('response')}>
                  <Button type="outline" loading={uploadingRole === 'response'}>上传响应文件</Button>
                </Upload>
                {renderUploadedInfo(responseFile, 'response')}
              </div>
              <div>
                <Typography.Text>规范条目</Typography.Text>
                <Space direction="vertical" size={4} style={{ width: '100%' }}>
                  <Space>
                    <Button type="outline" onClick={handleOpenRuleModal}>
                      选择规范条目
                    </Button>
                    {selectedRuleIds.length ? (
                      <Button type="text" onClick={handleClearSelectedRules}>
                        清空已选
                      </Button>
                    ) : null}
                  </Space>
                  {selectedRuleIds.length ? (
                    <Space wrap>
                      {selectedRuleIds.map((id) => {
                        const info = ruleDictionary[id]
                        const label = info
                          ? `${info.rule_id ? `${info.rule_id} · ` : ''}${info.title || '未命名条目'}`
                          : id
                        return (
                          <Tag
                            key={id}
                            closable
                            onClose={(event) => {
                              event?.preventDefault?.()
                              handleRuleRemove(id)
                            }}
                          >
                            {label}
                          </Tag>
                        )
                      })}
                    </Space>
                  ) : (
                    <Typography.Text type="secondary">请选择至少一条规范条目后再提交任务</Typography.Text>
                  )}
                </Space>
              </div>
              <Button type="primary" disabled={!canSubmit} loading={submitting} onClick={handleSubmit}>
                提交任务
              </Button>
            </Space>
          </Card>

          <Card
            title="任务列表"
            className="compliance-card"
            loading={loading}
            extra={
              <Button icon={<IconRefresh />} type="text" onClick={() => loadJobs(pagination.page, pagination.pageSize)}>
                刷新
              </Button>
            }
          >
            <div className="job-list">
              {jobs.map((job) => (
                <Card
                  key={job.job_id}
                  className={job.job_id === currentJobId ? 'job-item active' : 'job-item'}
                  onClick={() => handleSelectJob(job.job_id)}
                >
                  <Space direction="vertical" size={4} style={{ width: '100%' }}>
                    <Space align="center" style={{ width: '100%', justifyContent: 'space-between', display: 'flex' }}>
                      <Typography.Text style={{ fontWeight: 600 }}>任务 #{job.job_id}</Typography.Text>
                      <StatusTag status={job.status} />
                    </Space>
                    <Progress percent={job.progress} size="mini" status={job.status === 'failed' ? 'error' : 'normal'} />
                    {job.llm_model && (
                      <Typography.Text type="secondary">模型：{job.llm_model}</Typography.Text>
                    )}
                    {renderSummaryTags(job.analysis_summary)}
                    <Typography.Text type="secondary">创建时间：{job.created_at}</Typography.Text>
                    <Space>
                      <Button size="mini" type="text" onClick={(e) => { e.stopPropagation(); handleRetry(job.job_id) }}>重试</Button>
                      <Button size="mini" type="text" onClick={(e) => { e.stopPropagation(); handleDownload(job.job_id, 'pdf') }}>报告</Button>
                    </Space>
                  </Space>
                </Card>
              ))}
            </div>
            <Pagination
              size="small"
              current={pagination.page}
              pageSize={pagination.pageSize}
              total={pagination.total}
              onChange={(page, pageSize) => loadJobs(page, pageSize)}
              className="compliance-pagination"
            />
          </Card>
        </Col>

        <Col span={17} className="compliance-right">
          <Card title="任务详情" className="compliance-card" loading={issueLoading}>
            {currentDetail ? (
              <Space direction="vertical" size="large" style={{ width: '100%' }}>
                <Space>
                  <Button icon={<IconCloudDownload />} onClick={() => handleDownload(currentDetail.job.job_id, 'json')}>
                    下载 JSON
                  </Button>
                  <Button icon={<IconCloudDownload />} onClick={() => handleDownload(currentDetail.job.job_id, 'csv')}>
                    下载 CSV
                  </Button>
                  <Button icon={<IconCloudDownload />} onClick={() => handleDownload(currentDetail.job.job_id, 'pdf')}>
                    下载 PDF
                  </Button>
                </Space>
                <Typography.Text type="secondary">
                  模型：{currentDetail.job.llm_model || '未记录'} · 调用成本：{formatLLMCost(currentDetail.job.llm_cost)}
                </Typography.Text>
                {renderSummaryTags(currentDetail.job.analysis_summary)}
                <Tabs
                  activeTab={activeHighlightRole}
                  onChange={(key) => {
                    const role = key as 'tender' | 'response'
                    setActiveHighlightRole(role)
                    loadHighlights(currentDetail.job.job_id, role)
                  }}
                >
                  <TabPane key="response" title="响应高亮">
                    <HighlightView data={activeHighlightRole === 'response' ? highlights : []} loading={highlightLoading && activeHighlightRole === 'response'} />
                  </TabPane>
                  <TabPane key="tender" title="比选高亮">
                    <HighlightView data={activeHighlightRole === 'tender' ? highlights : []} loading={highlightLoading && activeHighlightRole === 'tender'} />
                  </TabPane>
                </Tabs>
                <IssueList issues={currentDetail.issues} loading={issueLoading} />
              </Space>
            ) : (
              <Typography.Text type="secondary">请选择左侧任务查看详情</Typography.Text>
            )}
          </Card>
        </Col>
      </Row>
      <Modal
        title="选择规范条目"
        visible={ruleModalVisible}
        onCancel={() => setRuleModalVisible(false)}
        onOk={handleRuleModalConfirm}
        okText="确认选择"
        cancelText="取消"
        style={{ width: 880 }}
        maskClosable={false}
        unmountOnExit
      >
        <Space align="center" style={{ width: '100%', marginBottom: 12 }}>
          <Input.Search
            allowClear
            placeholder="输入关键字过滤规范"
            value={ruleModalKeyword}
            onChange={(value) => {
              setRuleModalKeyword(value)
              if (!value.trim()) {
                loadRuleOptions('', true)
              }
            }}
            onSearch={handleRuleSearch}
            style={{ flex: 1 }}
          />
          <Button onClick={handleSelectAllRules}>全选当前</Button>
          <Button onClick={handleClearModalSelection}>清空选择</Button>
        </Space>
        <Table
          rowKey="id"
          size="small"
          loading={ruleModalLoading}
          columns={ruleModalColumns}
          data={ruleModalData}
          pagination={false}
          rowSelection={{
            type: 'checkbox',
            selectedRowKeys: ruleModalSelectedKeys,
            onChange: (keys) => setRuleModalSelectedKeys(keys as string[]),
          }}
          scroll={{ y: 360 }}
        />
      </Modal>
      <Modal
        title={rulePreview?.title || rulePreview?.rule_id || '规范原文'}
        visible={!!rulePreview}
        footer={null}
        onCancel={() => setRulePreview(null)}
        style={{ width: 720 }}
        unmountOnExit
        maskClosable={true}
      >
        <Typography.Paragraph style={{ whiteSpace: 'pre-wrap', maxHeight: '60vh', overflowY: 'auto' }}>
          {rulePreview?.content || rulePreview?.content_preview || '暂无内容'}
        </Typography.Paragraph>
      </Modal>
    </div>
  )
}

function StatusTag({ status }: { status: string }) {
  const meta = getStatusMeta(status)
  return <Tag color={meta.color}>{meta.label}</Tag>
}

function IssueList({ issues, loading }: { issues: ComplianceIssue[]; loading: boolean }) {
  if (loading) {
    return <Typography.Text type="secondary">加载中...</Typography.Text>
  }
  const requirementIssues = issues.filter((issue) => issue.source_type === 'tender_requirement')
  const ruleIssues = issues.filter((issue) => (issue.source_type ?? 'rule') !== 'tender_requirement')

  const renderIssueCard = (issue: ComplianceIssue) => {
    const scoreText = Number.isFinite(issue.match_score)
      ? `${(issue.match_score * 100).toFixed(1)}%`
      : '未知'
    const sourceLabel = issue.source_type === 'tender_requirement' ? '比选要求' : '规范条目'
    const title = issue.source_type === 'tender_requirement'
      ? issue.requirement_name || issue.rule_title
      : issue.rule_title
    const requirementDesc = issue.required_content
    return (
      <Card key={issue.id} className="issue-item" bordered>
        <Space direction="vertical" size={6} style={{ width: '100%' }}>
          <Space align="center" size={8}>
            <Typography.Text style={{ fontWeight: 600 }}>{title}</Typography.Text>
            <StatusTag status={issue.status} />
          </Space>
          <Typography.Text type="secondary">来源：{sourceLabel}</Typography.Text>
          <Typography.Text type="secondary">要求：{requirementDesc}</Typography.Text>
          <Typography.Paragraph style={{ marginBottom: 0 }}>
            响应文件：{issue.response_excerpt || '未匹配'}
          </Typography.Paragraph>
          <Typography.Text type="secondary">置信度：{scoreText}</Typography.Text>
          {issue.gap && <Typography.Text style={{ color: '#f53f3f' }}>差距：{issue.gap}</Typography.Text>}
          {issue.remark && <Typography.Text type="secondary">备注：{issue.remark}</Typography.Text>}
          {issue.llm_advice && <Typography.Text type="warning">整改建议：{issue.llm_advice}</Typography.Text>}
          {issue.llm_model && <Typography.Text type="secondary">判定模型：{issue.llm_model}</Typography.Text>}
        </Space>
      </Card>
    )
  }

  return (
    <Space direction="vertical" size="medium" style={{ width: '100%' }}>
      {requirementIssues.length > 0 && (
        <div>
          <Typography.Title heading={6} style={{ marginBottom: 8 }}>
            比选要求对比
          </Typography.Title>
          <Space direction="vertical" size="medium" style={{ width: '100%' }}>
            {requirementIssues.map(renderIssueCard)}
          </Space>
        </div>
      )}
      {ruleIssues.length > 0 && (
        <div>
          <Typography.Title heading={6} style={{ marginBottom: 8 }}>
            规范条目对比
          </Typography.Title>
          <Space direction="vertical" size="medium" style={{ width: '100%' }}>
            {ruleIssues.map(renderIssueCard)}
          </Space>
        </div>
      )}
    </Space>
  )
}

function HighlightView({ data, loading }: { data: ComplianceHighlight[]; loading: boolean }) {
  if (loading) {
    return <Typography.Text type="secondary">加载中...</Typography.Text>
  }
  if (!data.length) {
    return <Typography.Text type="secondary">暂无高亮数据</Typography.Text>
  }
  return (
    <Space direction="vertical" size="medium" style={{ width: '100%' }}>
      {data.map((item) => (
        <Card key={item.id} className="highlight-item" bordered>
          <Space direction="vertical" size={4}>
            <Space align="center" size={8}>
              <Tag color={item.file_role === 'response' ? 'green' : 'arcoblue'}>
                {item.file_role === 'response' ? '响应文件' : '比选文件'}
              </Tag>
              <Typography.Text type="secondary">
                偏移：
                {item.offset_start >= 0 && item.offset_end >= 0 ? `${item.offset_start} - ${item.offset_end}` : '未定位'}
              </Typography.Text>
            </Space>
            <Typography.Paragraph type={item.offset_start < 0 || item.offset_end < 0 ? 'error' : undefined}>
              {item.text || '（原文缺失）'}
            </Typography.Paragraph>
          </Space>
        </Card>
      ))}
    </Space>
  )
}

function formatLLMCost(cost?: number) {
  if (typeof cost !== 'number' || Number.isNaN(cost)) {
    return '未统计'
  }
  return `¥${cost.toFixed(4)}`
}

function renderSummaryTags(summary?: string) {
  const entries = parseSummary(summary)
  if (!entries.length) {
    return null
  }
  return (
    <Space wrap size={4}>
      {entries.map(({ status, count }) => {
        const meta = getStatusMeta(status)
        return (
          <Tag key={status} color={meta.color}>
            {meta.label}：{count}
          </Tag>
        )
      })}
    </Space>
  )
}

function parseSummary(summary?: string): Array<{ status: string; count: number }> {
  if (!summary) {
    return []
  }
  return summary
    .split(',')
    .map((part) => part.trim())
    .filter(Boolean)
    .map((part) => {
      const [status, count] = part.split(':')
      return {
        status: status?.trim() ?? '',
        count: Number(count ?? 0) || 0,
      }
    })
    .filter((item) => Boolean(item.status))
}
