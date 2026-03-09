import React from 'react'
import {createRoot} from 'react-dom/client'
import './style.css'
import App from './App'

// 过滤 antd 的 findDOMNode 警告
const originalError = console.error
console.error = (...args) => {
  if (
    typeof args[0] === 'string' &&
    args[0].includes('findDOMNode')
  ) {
    return
  }
  originalError.call(console, ...args)
}

const container = document.getElementById('root')

const root = createRoot(container!)

root.render(
    <App/>
)
