import { createRouter, createWebHistory } from 'vue-router'

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: '/',
      name: 'home',
      component: () => import('../views/HomeView.vue'),
    },
    {
      path: '/holes/:hole',
      name: 'hole',
      component: () => import('../views/HoleView.vue'),
      props: (route) => ({ hole: Number(route.params.hole) }),
    },
  ],
})

export default router
