#ifndef _PROJ_GO_H_
#define _PROJ_GO_H_

#include <proj.h>
#include <stdlib.h>

void trans(PJ *pj, PJ_DIRECTION direction, double u1, double v1, double w1, double t1, double *u2, double *v2, double *w2, double *t2);
PJ_COORD uvwt(double u, double v, double w, double t);

#endif
