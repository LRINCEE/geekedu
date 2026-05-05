import React, { useEffect, useState } from 'react'
import { Card, Row, Col, Pagination, Spin, Empty, Tag } from 'antd'
import { useNavigate } from 'react-router-dom'
import { listCourses } from '../../api/course'

interface CourseItem {
  id: number
  title: string
  description: string
  price: number
  cover_url: string
  created_at: string
}

const CourseList: React.FC = () => {
  const navigate = useNavigate()
  const [courses, setCourses] = useState<CourseItem[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [loading, setLoading] = useState(false)

  const fetchCourses = async (p: number) => {
    setLoading(true)
    try {
      const res: any = await listCourses(p, 12)
      setCourses(res.data.courses || [])
      setTotal(res.data.total || 0)
    } catch {
      setCourses([])
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    fetchCourses(page)
  }, [page])

  return (
    <div>
      <h2>课程列表</h2>
      <Spin spinning={loading}>
        {courses.length === 0 && !loading ? (
          <Empty description="暂无课程" />
        ) : (
          <>
            <Row gutter={[16, 16]}>
              {courses.map((course) => (
                <Col xs={24} sm={12} md={8} lg={6} key={course.id}>
                  <Card
                    hoverable
                    cover={
                      course.cover_url ? (
                        <img
                          alt={course.title}
                          src={course.cover_url}
                          style={{ height: 180, objectFit: 'cover' }}
                        />
                      ) : (
                        <div
                          style={{
                            height: 180,
                            background: '#f0f0f0',
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            color: '#999',
                          }}
                        >
                          暂无封面
                        </div>
                      )
                    }
                    onClick={() => navigate(`/courses/${course.id}`)}
                  >
                    <Card.Meta
                      title={course.title}
                      description={
                        <div>
                          <div style={{ marginBottom: 8, color: '#666', height: 40, overflow: 'hidden' }}>
                            {course.description || '暂无简介'}
                          </div>
                          <Tag color="orange">¥{course.price}</Tag>
                        </div>
                      }
                    />
                  </Card>
                </Col>
              ))}
            </Row>
            <div style={{ textAlign: 'center', marginTop: 24 }}>
              <Pagination current={page} total={total} pageSize={12} onChange={setPage} />
            </div>
          </>
        )}
      </Spin>
    </div>
  )
}

export default CourseList
