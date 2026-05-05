import React from 'react'
import { Outlet, useNavigate, Link } from 'react-router-dom'
import { Layout, Menu, Button, Space, message } from 'antd'
import {
  BookOutlined,
  PlusOutlined,
  UploadOutlined,
  LoginOutlined,
  LogoutOutlined,
} from '@ant-design/icons'
import { isLoggedIn, isAdmin, getUserInfo, logout } from '../utils/token'

const { Header, Content, Footer } = Layout

const AppLayout: React.FC = () => {
  const navigate = useNavigate()
  const loggedIn = isLoggedIn()
  const admin = isAdmin()
  const userInfo = getUserInfo()

  const handleLogout = () => {
    logout()
    message.success('已退出登录')
    navigate('/login')
  }

  const menuItems = [
    { key: 'courses', icon: <BookOutlined />, label: <Link to="/courses">课程列表</Link> },
    ...(admin
      ? [
          { key: 'create-course', icon: <PlusOutlined />, label: <Link to="/admin/course">发布课程</Link> },
          { key: 'upload-video', icon: <UploadOutlined />, label: <Link to="/admin/video">上传视频</Link> },
        ]
      : []),
  ]

  return (
    <Layout style={{ minHeight: '100vh' }}>
      <Header style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', flex: 1 }}>
          <div
            style={{ color: '#fff', fontSize: 20, fontWeight: 'bold', marginRight: 40, cursor: 'pointer' }}
            onClick={() => navigate('/courses')}
          >
            GeekEdu
          </div>
          <Menu theme="dark" mode="horizontal" items={menuItems} style={{ flex: 1 }} />
        </div>
        <Space>
          {loggedIn ? (
            <>
              <span style={{ color: '#fff' }}>
                {userInfo?.username} ({admin ? '管理员' : '学员'})
              </span>
              <Button type="text" icon={<LogoutOutlined />} onClick={handleLogout} style={{ color: '#fff' }}>
                退出
              </Button>
            </>
          ) : (
            <Button type="text" icon={<LoginOutlined />} onClick={() => navigate('/login')} style={{ color: '#fff' }}>
              登录
            </Button>
          )}
        </Space>
      </Header>
      <Content style={{ padding: '24px 48px' }}>
        <Outlet />
      </Content>
      <Footer style={{ textAlign: 'center' }}>GeekEdu Online Learning Platform</Footer>
    </Layout>
  )
}

export default AppLayout
