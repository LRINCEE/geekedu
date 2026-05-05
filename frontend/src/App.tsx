import React from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { ConfigProvider } from 'antd'
import zhCN from 'antd/locale/zh_CN'
import AppLayout from './components/Layout'
import Login from './pages/Login'
import Register from './pages/Register'
import CourseList from './pages/CourseList'
import CourseDetail from './pages/CourseDetail'
import Player from './pages/Player'
import CreateCourse from './pages/Admin/CreateCourse'
import UploadVideo from './pages/Admin/UploadVideo'

const App: React.FC = () => {
  return (
    <ConfigProvider locale={zhCN}>
      <BrowserRouter>
        <Routes>
          <Route path="/login" element={<Login />} />
          <Route path="/register" element={<Register />} />
          <Route path="/" element={<AppLayout />}>
            <Route index element={<Navigate to="/courses" replace />} />
            <Route path="courses" element={<CourseList />} />
            <Route path="courses/:id" element={<CourseDetail />} />
            <Route path="player/:videoId" element={<Player />} />
            <Route path="admin/course" element={<CreateCourse />} />
            <Route path="admin/video" element={<UploadVideo />} />
          </Route>
        </Routes>
      </BrowserRouter>
    </ConfigProvider>
  )
}

export default App
