import request from './request'
import axios from 'axios'

export function listCourses(page: number = 1, pageSize: number = 10) {
  return request.get('/courses', { params: { page, page_size: pageSize } })
}

export function getCourse(courseId: number) {
  return request.get(`/courses/${courseId}`)
}

export interface CreateCoursePayload {
  title: string
  description?: string
  price: number
  cover_key?: string
}

export function getCoverUploadURL(filename: string) {
  return request.get('/courses/cover/upload_url', { params: { filename } })
}

export async function uploadCover(uploadUrl: string, file: File): Promise<void> {
  await axios.put(uploadUrl, file, {
    headers: {
      // Keep this aligned with backend OSS SignURL, which signs an empty Content-Type.
      'Content-Type': '',
    },
  })
}

export function createCourse(payload: CreateCoursePayload) {
  return request.post('/courses', payload, {
    headers: { 'Content-Type': 'application/json' },
  })
}
