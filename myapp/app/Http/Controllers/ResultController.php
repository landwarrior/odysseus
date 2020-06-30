<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\MstProcess;
use App\TrnProject;
use App\TrnProjectDetailHr;
use App\TrnHrResult;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use App\Http\Requests\ResultRequest;

class ResultController extends Controller
{
    /**
     * Create a new controller instance.
     *
     * @return void
     */
    public function __construct()
    {
        $this->middleware('auth');
    }

    public function index()
    {
        $user = Auth::user();
        $targets = TrnProjectDetailHr::where('hr_cd', $user->hr_cd)->get();
        $p = [];
        foreach ($targets as $target) {
            $p[] = $target->project_no;
        }
        $projects = TrnProject::whereIn('project_no', $p)->get();
        return view('result.index', ['projects' => $projects]);
    }

    public function edit(Request $request, $project_no)
    {
        $user = Auth::user();
        $this_month = date('Y-m');
        $query_params = $request->query();
        if (!empty($query_params['date'])) {
            $this_month = $query_params['date'];
        }
        $last_month = date('Y-m', strtotime("{$this_month}-01  -1 month"));
        $next_month = date('Y-m', strtotime("{$this_month}-01  +1 month"));
        $processes = MstProcess::all();
        $process_map = [];
        foreach ($processes as $process) {
            $process_map[$process->process_id] = $process->name;
        }
        $targets = TrnProjectDetailHr::where('hr_cd', $user->hr_cd)
                   ->where('project_no', $project_no)->get();
        $hrResults = TrnHrResult::where('hr_cd', $user->hr_cd)
                     ->where('target_date', '>=', "{$this_month}-01")
                     ->where('target_date', '<', "{$next_month}-01")
                     ->orderBy('target_date', 'asc')
                     ->orderBy('process_id', 'asc')->get();
        $results = [];
        foreach ($hrResults as $data) {
            if (!isset($results[$data->target_date->format('Y/m/d')])) {
                $results[$data->target_date->format('Y/m/d')] = [];
            }
            $results[$data->target_date->format('Y/m/d')][$data->process_id] = $data->result_hour;
        }

        return view(
            'result.edit',
            [
                'project_no' => $project_no,
                'targets' => $targets,
                'last_month' => $last_month,
                'next_month' => $next_month,
                'results' => $results,
                'process_map' => $process_map,
            ]
        );
    }

    public function update(ResultRequest $request, $project_no)
    {
        $user = Auth::user();
        $processes = MstProcess::all();
        DB::transaction(function () use ($request, $project_no, $user, $processes) {
            foreach ($processes as $process) {
                TrnHrResult::where('hr_cd', $user->hr_cd)
                             ->where('target_date', $request->target_date)
                             ->where('project_no', $project_no)
                             ->where('process_id', $process->process_id)->delete();
                if (isset($request->result[$process->process_id])) {
                    $hrResult = new TrnHrResult();
                    $hrResult->project_no = $project_no;
                    $hrResult->process_id = $process->process_id;
                    $hrResult->hr_cd = $user->hr_cd;
                    $hrResult->target_date = $request->target_date;
                    $hrResult->result_hour = $request->result[$process->process_id];
                    $hrResult->save();
                }
            }
        });

        return redirect('/result')->with('registered', '1');
    }
}
