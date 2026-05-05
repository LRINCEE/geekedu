import request from './request'

export function createOrder(courseId: number) {
  return request.post('/orders', { course_id: courseId })
}
