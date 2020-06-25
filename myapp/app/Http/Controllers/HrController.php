<?php

namespace App\Http\Controllers;

use Illuminate\Http\Request;
use App\MstHr;
use App\MstRole;
use App\MstHrUnitPrice;
use Illuminate\Support\Facades\Auth;
use Illuminate\Support\Facades\DB;
use Illuminate\Support\Facades\Hash;
use App\Http\Requests\HrRequest;

class HrController extends Controller
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
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }
        $humans = MstHr::all();
        return view('hr.index', ['humans' => $humans]);
    }

    public function create()
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }

        $human = new MstHr();
        $roles = MstRole::all();
        $hrPrices = new MstHrUnitPrice();
        return view(
            'hr.create',
            ['human' => $human, 'roles' => $roles, 'prices' => [$hrPrices]]
        );
    }

    public function store(HrRequest $request)
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }
        DB::transaction(function () use ($request) {
            $human = new MstHr();
            $human->hr_cd = $request->hr_cd;
            $human->user_name = $request->user_name;
            $human->name_kana = $request->name_kana;
            $human->password = Hash::make('12345678');
            $human->is_admin = $request->is_admin;
            $human->remarks = $request->remarks;
            $human->save();
            MstHrUnitPrice::where('hr_cd', $request->hr_cd)->delete();
            if ($request->prices) {
                foreach ($request->prices as $price) {
                    $hrPrice = new MstHrUnitPrice();
                    $hrPrice->hr_cd = $request->hr_cd;
                    $hrPrice->role_id = $price['role_id'];
                    $hrPrice->price = $price['price'];
                    $hrPrice->from_date = $price['from_date'];
                    $hrPrice->save();
                }
            }
        });

        return redirect('/hr')->with('registered', '1');
    }

    public function edit($hr_cd)
    {
        $user = Auth::user();
        if (!$user->is_admin) {
            return redirect(route('home'))->with('not_admin', '1');
        }

        $human = MstHr::findOrFail($hr_cd);
        $roles = MstRole::all();
        $prices = MstHrUnitPrice::where('hr_cd', $hr_cd)->get();
        return view(
            'hr.edit',
            ['human' => $human, 'roles' => $roles, 'prices' => $prices]
        );
    }

    public function update(HrRequest $request, $hr_cd)
    {
        DB::transaction(function () use ($request, $hr_cd) {
            $human = MstHr::findOrFail($hr_cd);
            $human->user_name = $request->user_name;
            $human->name_kana = $request->name_kana;
            $human->is_admin = $request->is_admin;
            $human->remarks = $request->remarks;
            $human->save();
            MstHrUnitPrice::where('hr_cd', $hr_cd)->delete();
            if ($request->prices) {
                foreach ($request->prices as $price) {
                    $hrPrice = new MstHrUnitPrice();
                    $hrPrice->hr_cd = $hr_cd;
                    $hrPrice->role_id = $price['role_id'];
                    $hrPrice->price = $price['price'];
                    $hrPrice->from_date = $price['from_date'];
                    $hrPrice->save();
                }
            }
        });

        return redirect('/hr')->with('registered', '1');
    }

    public function delete(Request $request, $hr_cd)
    {
        DB::transaction(function () use ($hr_cd) {
            MstHrUnitPrice::where('hr_cd', $hr_cd)->delete();
            MstHr::find($hr_cd)->delete();
        });
        return redirect('/hr')->with('deleted', '1');
    }
}
