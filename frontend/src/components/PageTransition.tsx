import React from 'react'
import { useLocation } from 'react-router-dom'
import './PageTransition.css'

interface PageTransitionProps {
  children: React.ReactNode
}

const PageTransition: React.FC<PageTransitionProps> = ({ children }) => {
  const location = useLocation()

  return (
    <div key={location.pathname} className="page-transition">
      {children}
    </div>
  )
}

export default PageTransition
