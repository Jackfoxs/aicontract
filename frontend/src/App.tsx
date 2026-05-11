import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import BasicLayout from './layouts/BasicLayout'
import ArticleList from './pages/Article/ArticleList'
import ArticleDetail from './pages/Article/ArticleDetail'
import ChunkEditor from './pages/Article/ChunkEditor'
import Chat from './pages/Chat'
import Search from './pages/Search'
// 采购需求模块
import RequirementInput from './pages/Procurement/RequirementInput'
import ParameterGenerate from './pages/Procurement/ParameterGenerate'
import RequirementList from './pages/Procurement/RequirementList'
// 合同审核模块
import ContractUpload from './pages/Contract/ContractUpload'
import ContractReview from './pages/Contract/ContractReview'
import ContractList from './pages/Contract/ContractList'
import CompliancePage from './pages/Compliance'
import '@arco-design/web-react/dist/css/arco.css'
import './App.css'

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<BasicLayout />}>
          <Route index element={<Navigate to="/articles" replace />} />
          <Route path="articles" element={<ArticleList />} />
          <Route path="articles/:id" element={<ArticleDetail />} />
          <Route path="articles/:id/chunks" element={<ChunkEditor />} />
          <Route path="chat" element={<Chat />} />
          <Route path="search" element={<Search />} />
          
          {/* 采购需求模块 */}
          <Route path="procurement/input" element={<RequirementInput />} />
          <Route path="procurement/generate/:id" element={<ParameterGenerate />} />
          <Route path="procurement/list" element={<RequirementList />} />
          
          {/* 合同审核模块 */}
          <Route path="contract/upload" element={<ContractUpload />} />
          <Route path="contract/review/:id" element={<ContractReview />} />
          <Route path="contract/list" element={<ContractList />} />

          {/* 合规比对模块 */}
          <Route path="compliance" element={<CompliancePage />} />
        </Route>
      </Routes>
    </BrowserRouter>
  )
}

export default App

