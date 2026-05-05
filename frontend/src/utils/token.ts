export function getToken(): string | null {
  return localStorage.getItem('token')
}

export function setToken(token: string): void {
  localStorage.setItem('token', token)
}

export function removeToken(): void {
  localStorage.removeItem('token')
}

export function getUserInfo(): { userId: number; role: number; username: string } | null {
  const info = localStorage.getItem('userInfo')
  if (!info) return null
  return JSON.parse(info)
}

export function setUserInfo(info: { userId: number; role: number; username: string }): void {
  localStorage.setItem('userInfo', JSON.stringify(info))
}

export function removeUserInfo(): void {
  localStorage.removeItem('userInfo')
}

export function isAdmin(): boolean {
  const info = getUserInfo()
  return info?.role === 1
}

export function isLoggedIn(): boolean {
  return !!getToken()
}

export function logout(): void {
  removeToken()
  removeUserInfo()
}
