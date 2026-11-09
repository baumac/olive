import { createRouter, createWebHistory } from 'vue-router'
import ErrorPage from '@/components/app/ErrorPage.vue'
import { signInPath } from '@/composables/useAuth'
import { signedIn } from '@/lib/session'
import { safeRedirect } from '@/lib/validation'

declare module 'vue-router' {
  interface RouteMeta {
    // requiresAuth sends a signed out person to the sign in page first.
    requiresAuth?: boolean
    // guestOnly sends a signed in person to the redirect path.
    guestOnly?: boolean
  }
}

export const router = createRouter({
  history: createWebHistory(),
  scrollBehavior: (_to, _from, saved) => saved ?? { top: 0 },
  routes: [
    { path: '/', redirect: '/restaurants' },
    { path: '/restaurants', component: () => import('@/pages/RestaurantsPage.vue') },
    { path: '/my-reviews', component: () => import('@/pages/MyReviewsPage.vue'), meta: { requiresAuth: true } },
    {
      path: '/restaurants/new',
      component: () => import('@/pages/RestaurantFormPage.vue'),
      meta: { requiresAuth: true },
    },
    { path: '/restaurants/:restaurantId', component: () => import('@/pages/RestaurantPage.vue'), props: true },
    {
      path: '/restaurants/:restaurantId/edit',
      component: () => import('@/pages/RestaurantFormPage.vue'),
      props: true,
      meta: { requiresAuth: true },
    },
    {
      path: '/restaurants/:restaurantId/reviews/new',
      component: () => import('@/pages/ReviewFormPage.vue'),
      props: true,
      meta: { requiresAuth: true },
    },
    {
      path: '/restaurants/:restaurantId/reviews/:reviewId',
      component: () => import('@/pages/ReviewPage.vue'),
      props: true,
    },
    {
      path: '/restaurants/:restaurantId/reviews/:reviewId/edit',
      component: () => import('@/pages/ReviewFormPage.vue'),
      props: true,
      meta: { requiresAuth: true },
    },
    { path: '/sign-in', component: () => import('@/pages/SignInPage.vue'), meta: { guestOnly: true } },
    { path: '/sign-up', component: () => import('@/pages/SignUpPage.vue'), meta: { guestOnly: true } },
    { path: '/:pathMatch(.*)*', component: ErrorPage, props: { status: 404 } },
  ],
})

router.beforeEach((to) => {
  if (to.meta.requiresAuth && !signedIn.value) {
    return signInPath(to.fullPath)
  }
  if (to.meta.guestOnly && signedIn.value) {
    return safeRedirect(to.query.redirect)
  }
  return true
})
