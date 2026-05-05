import React, { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Card, Button, List, Tag, message, Spin, Descriptions } from 'antd'
import { PlayCircleOutlined, ShoppingCartOutlined } from '@ant-design/icons'
import { getCourse } from '../../api/course'
import { createOrder } from '../../api/order'
import { isLoggedIn } from '../../utils/token'

interface VideoItem {
  id: number
  title: string
  sort_order: number
  course_id: number
}

interface CourseInfo {
  id: number
  title: string
  description: string
  price: number
  cover_url: string
  created_at: string
  teacher_id: number
}

const CourseDetail: React.FC = () => {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const [course, setCourse] = useState<CourseInfo | null>(null)
  const [videos, setVideos] = useState<VideoItem[]>([])
  const [loading, setLoading] = useState(true)
  const [buying, setBuying] = useState(false)

  useEffect(() => {
    const fetchCourse = async () => {
      try {
        const res: any = await getCourse(Number(id))
        setCourse(res.data.course)
        setVideos(res.data.videos || [])
      } catch (err: any) {
        message.error(err.message || '获取课程失败')
      } finally {
        setLoading(false)
      }
    }
    fetchCourse()
  }, [id])

  const handleBuy = async () => {
    if (!isLoggedIn()) {
      message.warning('请先登录')
      navigate('/login')
      return
    }
    setBuying(true)
    try {
      await createOrder(Number(id))
      message.success('购买成功！')
    } catch (err: any) {
      message.error(err.message || '购买失败')
    } finally {
      setBuying(false)
    }
  }

  const handlePlay = (videoId: number) => {
    if (!isLoggedIn()) {
      message.warning('请先登录')
      navigate('/login')
      return
    }
    navigate(`/player/${videoId}`)
  }

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />
  if (!course) return <div>课程不存在</div>

  return (
    <div style={{ maxWidth: 900, margin: '0 auto' }}>
      <Card
        cover={
          course.cover_url ? (
            <img alt={course.title} src={course.cover_url} style={{ maxHeight: 400, objectFit: 'cover' }} />
          ) : null
        }
      >
        <Descriptions title={course.title} column={1}>
          <Descriptions.Item label="价格">
            <Tag color="orange" style={{ fontSize: 16 }}>¥{course.price}</Tag>
          </Descriptions.Item>
          <Descriptions.Item label="简介">{course.description || '暂无简介'}</Descriptions.Item>
          <Descriptions.Item label="创建时间">{course.created_at}</Descriptions.Item>
        </Descriptions>
        <Button
          type="primary"
          size="large"
          icon={<ShoppingCartOutlined />}
          onClick={handleBuy}
          loading={buying}
          style={{ marginTop: 16 }}
        >
          购买课程
        </Button>
      </Card>

      <Card title="课程视频" style={{ marginTop: 16 }}>
        {videos.length === 0 ? (
          <div style={{ color: '#999' }}>暂无视频</div>
        ) : (
          <List
            dataSource={videos}
            renderItem={(video, index) => (
              <List.Item
                actions={[
                  <Button
                    type="link"
                    icon={<PlayCircleOutlined />}
                    onClick={() => handlePlay(video.id)}
                  >
                    播放
                  </Button>,
                ]}
              >
                <List.Item.Meta
                  title={`${index + 1}. ${video.title}`}
                />
              </List.Item>
            )}
          />
        )}
      </Card>
    </div>
  )
}

export default CourseDetail
