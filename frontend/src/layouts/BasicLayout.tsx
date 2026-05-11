import { useState } from 'react'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { Layout, Menu } from '@arco-design/web-react'
import {
  IconFile,
  IconMessage,
  IconSearch,
  IconRobot,
  IconSafe,
  IconList,
  IconEdit,
  IconUpload,
  IconSync,
} from '@arco-design/web-react/icon'
import './BasicLayout.css'

const { Sider, Header, Content } = Layout
const MenuItem = Menu.Item
const SubMenu = Menu.SubMenu

export default function BasicLayout() {
  const navigate = useNavigate()
  const location = useLocation()
  const [collapsed, setCollapsed] = useState(false)

  const handleMenuClick = (key: string) => {
    navigate(key)
  }

  // 获取当前激活的菜单项
  const getSelectedKeys = () => {
    const path = location.pathname
    // 如果在子页面，高亮父菜单
    if (path.startsWith('/procurement/')) {
      return [path, '/procurement']
    }
    if (path.startsWith('/contract/')) {
      return [path, '/contract']
    }
    if (path.startsWith('/compliance')) {
      return [path]
    }
    return [path]
  }

  return (
    <Layout className="basic-layout">
      <Sider
        collapsed={collapsed}
        onCollapse={setCollapsed}
        collapsible
        trigger={null}
        breakpoint="xl"
        style={{ height: '100vh' }}
      >
        <div className="logo">
          {!collapsed && <span>AI采购管理</span>}
        </div>
        <Menu
          selectedKeys={getSelectedKeys()}
          onClickMenuItem={handleMenuClick}
          style={{ width: '100%' }}
          defaultOpenKeys={['/procurement', '/contract']}
        >
          {/* 知识库管理 */}
          <MenuItem key="/articles">
            <IconFile />
          规范管理
          </MenuItem>
          <MenuItem key="/chat">
            <IconMessage />
            智能问答
          </MenuItem>
          <MenuItem key="/search">
            <IconSearch />
            文档搜索
          </MenuItem>

          {/* 采购管理 */}
          <SubMenu
            key="/procurement"
            title={
              <>
                <IconRobot />
                采购管理
              </>
            }
          >
            <MenuItem key="/procurement/input">
              <IconEdit />
              需求分析
            </MenuItem>
            <MenuItem key="/procurement/list">
              <IconList />
              需求列表
            </MenuItem>
          </SubMenu>

          {/* 合同审核 */}
          <SubMenu
            key="/contract"
            title={
              <>
                <IconSafe />
                合同审核
              </>
            }
          >
            <MenuItem key="/contract/upload">
              <IconUpload />
              上传审核
            </MenuItem>
            <MenuItem key="/contract/list">
              <IconList />
              审核记录
            </MenuItem>
          </SubMenu>

          <MenuItem key="/compliance">
            <IconSync />
            响应合规比对
          </MenuItem>
        </Menu>
      </Sider>
      <Layout>
        <Header className="layout-header">
          <div className="header-title">AI智能采购管理系统</div>
        </Header>
        <Content className="layout-content">
          <Outlet />
        </Content>
      </Layout>
    </Layout>
  )
}

