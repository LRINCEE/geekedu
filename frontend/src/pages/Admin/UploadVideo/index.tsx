import React, { useEffect, useState } from 'react'
import { Form, Input, Select, Button, Card, Progress, message } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import { listCourses } from '../../../api/course'
import { initMultipartUpload, completeMultipartUpload, uploadPart } from '../../../api/player'
import { isAdmin } from '../../../utils/token'

const PART_SIZE = 5 * 1024 * 1024 // 5MB per part

interface CourseOption {
  id: number
  title: string
}

const UploadVideo: React.FC = () => {
  const [courses, setCourses] = useState<CourseOption[]>([])
  const [uploading, setUploading] = useState(false)
  const [progress, setProgress] = useState(0)
  const [selectedFile, setSelectedFile] = useState<File | null>(null)
  const [form] = Form.useForm()

  useEffect(() => {
    const fetchCourses = async () => {
      try {
        const res: any = await listCourses(1, 100)
        setCourses(
          (res.data.courses || []).map((c: any) => ({ id: c.id, title: c.title }))
        )
      } catch {
        // ignore
      }
    }
    fetchCourses()
  }, [])

  if (!isAdmin()) {
    return <div>无权限访问</div>
  }

  const handleFileSelect = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (file) {
      setSelectedFile(file)
    }
  }

  const handleUpload = async (values: { course_id: number; title: string }) => {
    if (!selectedFile) {
      message.error('请选择视频文件')
      return
    }

    setUploading(true)
    setProgress(0)

    try {
      const partCount = Math.ceil(selectedFile.size / PART_SIZE)

      // 1. Init multipart upload
      const initRes: any = await initMultipartUpload(
        values.course_id,
        values.title,
        selectedFile.name,
        partCount
      )

      const { upload_id, object_key, upload_urls } = initRes.data

      // 2. Upload each part
      const parts: { part_number: number; etag: string }[] = []

      for (let i = 0; i < partCount; i++) {
        const start = i * PART_SIZE
        const end = Math.min(start + PART_SIZE, selectedFile.size)
        const blob = selectedFile.slice(start, end)

        const etag = await uploadPart(upload_urls[i], blob)
        parts.push({ part_number: i + 1, etag })

        setProgress(Math.round(((i + 1) / partCount) * 100))
      }

      // 3. Complete multipart upload
      await completeMultipartUpload(values.course_id, upload_id, object_key, values.title, parts)

      message.success('视频上传成功')
      setSelectedFile(null)
      setProgress(0)
      form.resetFields()
    } catch (err: any) {
      message.error(err.message || '上传失败')
    } finally {
      setUploading(false)
    }
  }

  return (
    <div style={{ maxWidth: 600, margin: '0 auto' }}>
      <Card title="上传课程视频">
        <Form form={form} layout="vertical" onFinish={handleUpload}>
          <Form.Item name="course_id" label="选择课程" rules={[{ required: true, message: '请选择课程' }]}>
            <Select placeholder="请选择课程">
              {courses.map((c) => (
                <Select.Option key={c.id} value={c.id}>
                  {c.title}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="title" label="视频标题" rules={[{ required: true, message: '请输入视频标题' }]}>
            <Input placeholder="请输入视频标题" />
          </Form.Item>
          <Form.Item label="视频文件" required>
            <input type="file" accept="video/*" onChange={handleFileSelect} />
            {selectedFile && (
              <div style={{ marginTop: 8, color: '#666' }}>
                已选择: {selectedFile.name} ({(selectedFile.size / 1024 / 1024).toFixed(2)} MB)
              </div>
            )}
          </Form.Item>
          {uploading && <Progress percent={progress} style={{ marginBottom: 16 }} />}
          <Form.Item>
            <Button
              type="primary"
              htmlType="submit"
              loading={uploading}
              icon={<UploadOutlined />}
              block
            >
              {uploading ? '上传中...' : '开始上传'}
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}

export default UploadVideo
