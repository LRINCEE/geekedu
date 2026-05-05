import React, { useState } from 'react'
import { Form, Input, InputNumber, Upload, Button, Card, message } from 'antd'
import { UploadOutlined } from '@ant-design/icons'
import { useNavigate } from 'react-router-dom'
import { createCourse, getCoverUploadURL, uploadCover } from '../../../api/course'
import { isAdmin } from '../../../utils/token'

const CreateCourse: React.FC = () => {
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [fileList, setFileList] = useState<any[]>([])

  if (!isAdmin()) {
    return <div>无权限访问</div>
  }

  const onFinish = async (values: { title: string; description: string; price: number }) => {
    setLoading(true)
    try {
      let coverKey = ''
      if (fileList.length > 0 && fileList[0].originFileObj) {
        const coverFile = fileList[0].originFileObj as File
        const uploadRes: any = await getCoverUploadURL(coverFile.name)
        const { upload_url, cover_key } = uploadRes.data
        await uploadCover(upload_url, coverFile)
        coverKey = cover_key
      }

      await createCourse({
        title: values.title,
        description: values.description || '',
        price: values.price || 0,
        cover_key: coverKey,
      })
      message.success('课程创建成功')
      navigate('/courses')
    } catch (err: any) {
      message.error(err.message || '创建失败')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div style={{ maxWidth: 600, margin: '0 auto' }}>
      <Card title="发布新课程">
        <Form layout="vertical" onFinish={onFinish}>
          <Form.Item name="title" label="课程标题" rules={[{ required: true, message: '请输入课程标题' }]}>
            <Input placeholder="请输入课程标题" />
          </Form.Item>
          <Form.Item name="description" label="课程简介">
            <Input.TextArea rows={4} placeholder="请输入课程简介" />
          </Form.Item>
          <Form.Item name="price" label="课程价格(元)" rules={[{ required: true, message: '请输入价格' }]}>
            <InputNumber min={0} precision={2} style={{ width: '100%' }} placeholder="0.00" />
          </Form.Item>
          <Form.Item label="封面图">
            <Upload
              listType="picture"
              maxCount={1}
              fileList={fileList}
              beforeUpload={() => false}
              onChange={({ fileList }) => setFileList(fileList)}
            >
              <Button icon={<UploadOutlined />}>选择封面图</Button>
            </Upload>
          </Form.Item>
          <Form.Item>
            <Button type="primary" htmlType="submit" loading={loading} block>
              发布课程
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  )
}

export default CreateCourse
