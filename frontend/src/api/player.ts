import request from './request'
import axios from 'axios'

export function getPlayURL(videoId: number) {
  return request.get(`/player/${videoId}`)
}

export function initMultipartUpload(courseId: number, title: string, filename: string, partCount: number) {
  return request.post(`/courses/${courseId}/videos/init`, {
    title,
    filename,
    part_count: partCount,
  })
}

export function completeMultipartUpload(
  courseId: number,
  uploadId: string,
  objectKey: string,
  title: string,
  parts: { part_number: number; etag: string }[]
) {
  return request.post(`/courses/${courseId}/videos/complete`, {
    upload_id: uploadId,
    object_key: objectKey,
    title,
    parts,
  })
}

export async function uploadPart(url: string, data: Blob): Promise<string> {
  const resp = await axios.put(url, data, {
    headers: { 
      // 强制覆盖 axios 的默认行为，什么类型都不要带，与后端的签名保持绝对一致
      'Content-Type': '' 
    },
  })
  return resp.headers['etag'] || ''
}
