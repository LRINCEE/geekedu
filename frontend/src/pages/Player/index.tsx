import React, { useEffect, useState } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { Spin, message, Button, Alert } from 'antd'
import { ArrowLeftOutlined } from '@ant-design/icons'
import { getPlayURL } from '../../api/player'
import { isLoggedIn } from '../../utils/token'

const Player: React.FC = () => {
  const { videoId } = useParams<{ videoId: string }>()
  const navigate = useNavigate()
  const [playUrl, setPlayUrl] = useState<string>('')
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string>('')

  useEffect(() => {
    if (!isLoggedIn()) {
      message.warning('请先登录')
      navigate('/login')
      return
    }

    const fetchPlayURL = async () => {
      try {
        const res: any = await getPlayURL(Number(videoId))
        setPlayUrl(res.data.play_url)
      } catch (err: any) {
        setError(err.message || '获取播放地址失败')
      } finally {
        setLoading(false)
      }
    }
    fetchPlayURL()
  }, [videoId, navigate])

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />

  return (
    <div style={{ maxWidth: 900, margin: '0 auto' }}>
      <Button
        icon={<ArrowLeftOutlined />}
        onClick={() => navigate(-1)}
        style={{ marginBottom: 16 }}
      >
        返回
      </Button>

      {error ? (
        <Alert
          message="无法播放"
          description={error}
          type="error"
          showIcon
        />
      ) : (
        <div style={{ background: '#000', borderRadius: 8, overflow: 'hidden' }}>
          <video
            src={playUrl}
            controls
            autoPlay
            style={{ width: '100%', maxHeight: '70vh' }}
          >
            您的浏览器不支持视频播放
          </video>
        </div>
      )}
    </div>
  )
}

export default Player
