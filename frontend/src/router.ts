import { createRouter, createWebHashHistory } from 'vue-router'
import AppShell from '@/components/layout/AppShell.vue'
import DashboardView from '@/views/DashboardView.vue'
import ProfilesView from '@/views/ProfilesView.vue'
import LogsView from '@/views/LogsView.vue'
import SettingsView from '@/views/SettingsView.vue'
import SetupView from '@/views/SetupView.vue'

const router = createRouter({
  history: createWebHashHistory(),
  routes: [
    {
      path: '/',
      component: AppShell,
      children: [
        { path: '', name: 'dashboard', component: DashboardView, meta: { title: '总览' } },
        { path: 'profiles', name: 'profiles', component: ProfilesView, meta: { title: '连接' } },
        { path: 'logs', name: 'logs', component: LogsView, meta: { title: '日志' } },
        { path: 'settings', name: 'settings', component: SettingsView, meta: { title: '设置' } },
      ],
    },
    { path: '/setup', name: 'setup', component: SetupView, meta: { title: '开始使用' } },
    { path: '/:pathMatch(.*)*', redirect: '/' },
  ],
})

router.afterEach((to) => {
  const title = (to.meta.title as string) || 'NPC 客户端'
  document.title = title + ' · NPC 客户端'
})

export default router
